package apiservices

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

// capabilityEntry è il formato JSON esposto dall'endpoint GET /capabilities.
// Corrisponde al formato atteso dal discovery loader di app-fe.
// id:          ID UPPER_SNAKE locale (prefissato dal gateway con il proxy id)
// operationId: operationId originale Huma per Match() nel backend; omesso se uguale a id
// category:    "api" | "action_api"
type capabilityEntry struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Method      string `json:"method,omitempty"`
}

// capDefDoc è il documento cap-def per YAML/Mongo, allineato al formato ng-core-ui.
type capDefDoc struct {
	ID          string `json:"_id"`
	ET          string `json:"_et"`
	App         string `json:"app"`
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Method      string `json:"method,omitempty"`
	SysInfo     string `json:"sys_info"`
}

// actionCapabilities raccoglie le capability action_api registrate dall'applicazione.
var actionCapabilities []capabilityEntry

// RegisterActionCapability registra una capability action_api inclusa nella risposta
// di GET /capabilities. Chiamare durante l'inizializzazione dell'applicazione.
func RegisterActionCapability(id, description string) {
	actionCapabilities = append(actionCapabilities, capabilityEntry{
		ID:          id,
		Category:    "action_api",
		Description: description,
	})
}

// capabilitiesHandler serve GET /capabilities → JSON.
func capabilitiesHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := buildEntries(api)
		data, err := json.Marshal(entries)
		if err != nil {
			log.Error().Err(err).Msg("capabilities: marshal error")
			http.Error(w, `{"error":"internal_error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

// capabilitiesYAMLHandler serve GET /capabilities.yaml → YAML cap_defs + cap_groups.
func capabilitiesYAMLHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := buildEntries(api)
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toCapabilitiesYAML(entries)))
	}
}

// capabilitiesMongoHandler serve GET /acl.mongo.js → script replaceOne upsert per MongoDB.
func capabilitiesMongoHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := buildEntries(api)
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toCapabilitiesMongo(entries)))
	}
}

// capID costruisce l'_id strutturato per un capability entry: api:<appID>:<id>.
func capID(appID, id string) string {
	return fmt.Sprintf("cap:%s:api:%s", appID, strings.ToLower(id))
}

// toCapabilitiesYAML serializza le capability nel formato cap_defs + cap_groups
// allineato a ng-core-ui toRoutesYaml.
func toCapabilitiesYAML(entries []capabilityEntry) string {
	appID := core.AppName
	var sb strings.Builder

	sb.WriteString("cap_defs:\n")
	for _, e := range entries {
		fmt.Fprintf(&sb, "  - category: %s\n", e.Category)
		fmt.Fprintf(&sb, "    _id: %q\n", capID(appID, e.ID))
		fmt.Fprintf(&sb, "    app: %q\n", appID)
		if e.Description != "" {
			fmt.Fprintf(&sb, "    description: %q\n", e.Description)
		}
		if e.OperationID != "" {
			fmt.Fprintf(&sb, "    operationId: %q\n", e.OperationID)
		}
		if e.Endpoint != "" {
			fmt.Fprintf(&sb, "    endpoint: %q\n", e.Endpoint)
		}
		if e.Method != "" {
			fmt.Fprintf(&sb, "    method: %q\n", e.Method)
		}
	}

	groupID := fmt.Sprintf("grp:%s:ALL", appID)
	fmt.Fprintf(&sb, "cap_groups:\n  - _id: %q\n    description: \"\"\n    capabilities:\n", groupID)
	for _, e := range entries {
		fmt.Fprintf(&sb, "      - %q\n", capID(appID, e.ID))
	}

	return sb.String()
}

// toCapabilitiesMongo serializza le capability come script replaceOne upsert
// allineato a ng-core-ui toRoutesMongo.
func toCapabilitiesMongo(entries []capabilityEntry) string {
	appID := core.AppName
	var sb strings.Builder

	sb.WriteString("const COLLECTION = \"acl\";\n\n")

	for _, e := range entries {
		doc := capDefDoc{
			ID:          capID(appID, e.ID),
			ET:          "cap-def",
			App:         appID,
			Category:    e.Category,
			Description: e.Description,
			OperationID: e.OperationID,
			Endpoint:    e.Endpoint,
			Method:      e.Method,
			SysInfo:     "__SYS_INFO__",
		}
		raw, _ := json.MarshalIndent(doc, "    ", "    ")
		body := strings.ReplaceAll(
			string(raw),
			`"__SYS_INFO__"`,
			"{\n        status: \"active\",\n        created_at: new Date(),\n        modified_at: new Date()\n    }",
		)
		fmt.Fprintf(&sb,
			"db.getCollection(COLLECTION).replaceOne(\n    { _id: %s },\n    %s,\n    { upsert: true }\n)\n\n",
			jsonStr(capID(appID, e.ID)), body,
		)
	}

	// cap-group
	groupID := fmt.Sprintf("grp:%s:ALL", appID)
	var caps []string
	for _, e := range entries {
		caps = append(caps, fmt.Sprintf("        %s", jsonStr(capID(appID, e.ID))))
	}
	groupDoc := fmt.Sprintf(
		"{\n    _id: %s,\n    _et: \"cap-group\",\n    description: \"\",\n    capabilities: [\n%s\n    ],\n    sys_info: {\n        status: \"active\",\n        created_at: new Date(),\n        modified_at: new Date()\n    }\n}",
		jsonStr(groupID), strings.Join(caps, ",\n"),
	)
	fmt.Fprintf(&sb,
		"db.getCollection(COLLECTION).replaceOne(\n    { _id: %s },\n    %s,\n    { upsert: true }\n)\n",
		jsonStr(groupID), groupDoc,
	)

	return sb.String()
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// sqlStr wraps a string in single quotes with SQL-standard escaping (' → ”).
func sqlStr(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// capabilitiesSQLHandler serve GET /acl.sql → script INSERT upsert per SQL (PostgreSQL).
func capabilitiesSQLHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := buildEntries(api)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toCapabilitiesSQL(entries)))
	}
}

// toCapabilitiesSQL serializza le capability come script INSERT ... ON CONFLICT upsert
// per le tabelle opem_acl_cap_def, opem_acl_cap_group e opem_acl_cap_group_def.
// Sintassi: PostgreSQL / SQLite (ON CONFLICT ... DO UPDATE / DO NOTHING).
func toCapabilitiesSQL(entries []capabilityEntry) string {
	appID := core.AppName
	var sb strings.Builder

	sb.WriteString("-- SQL seed: opem_acl_cap_def + opem_acl_cap_group + opem_acl_cap_group_def\n")
	sb.WriteString("-- Sintassi: PostgreSQL / SQLite  (ON CONFLICT ... DO UPDATE / DO NOTHING)\n")
	sb.WriteString("-- Generato da GET /acl.sql — idempotente.\n\n")

	// ── cap_def ──────────────────────────────────────────────────────────────
	sb.WriteString("-- cap_defs\n")
	for _, e := range entries {
		id := capID(appID, e.ID)
		name := e.Description
		if name == "" {
			name = e.ID
		}
		fmt.Fprintf(&sb,
			"INSERT INTO opem_acl_cap_def (id, app, category, description, name, endpoint, icon, ord, menu, mapping, method, status)\n"+
				"VALUES (%s, %s, %s, %s, %s, %s, '', 0, FALSE, %s, %s, 'active')\n"+
				"ON CONFLICT (id) DO UPDATE SET\n"+
				"    app = EXCLUDED.app, category = EXCLUDED.category, description = EXCLUDED.description,\n"+
				"    name = EXCLUDED.name, endpoint = EXCLUDED.endpoint, mapping = EXCLUDED.mapping,\n"+
				"    method = EXCLUDED.method;\n\n",
			sqlStr(id), sqlStr(appID), sqlStr(e.Category), sqlStr(e.Description),
			sqlStr(name), sqlStr(e.Endpoint), sqlStr(e.OperationID), sqlStr(e.Method),
		)
	}

	// ── cap_group ─────────────────────────────────────────────────────────────
	groupID := fmt.Sprintf("grp:%s:ALL", appID)
	sb.WriteString("-- cap_group\n")
	fmt.Fprintf(&sb,
		"INSERT INTO opem_acl_cap_group (id, description, status)\n"+
			"VALUES (%s, '', 'active')\n"+
			"ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, status = EXCLUDED.status;\n\n",
		sqlStr(groupID),
	)

	// ── cap_group_def (junction) ──────────────────────────────────────────────
	sb.WriteString("-- cap_group_def\n")
	for _, e := range entries {
		fmt.Fprintf(&sb,
			"INSERT INTO opem_acl_cap_group_def (cap_group_id, cap_def_id) VALUES (%s, %s)\n"+
				"ON CONFLICT (cap_group_id, cap_def_id) DO NOTHING;\n",
			sqlStr(groupID), sqlStr(capID(appID, e.ID)),
		)
	}

	return sb.String()
}

// buildEntries costruisce le capabilityEntry dalle operazioni Huma + action_api registrate.
func buildEntries(api huma.API) []capabilityEntry {
	openapi := api.OpenAPI()

	type methodOp struct {
		method string
		op     *huma.Operation
	}

	var entries []capabilityEntry
	for path, item := range openapi.Paths {
		candidates := []methodOp{
			{"GET", item.Get},
			{"POST", item.Post},
			{"PUT", item.Put},
			{"DELETE", item.Delete},
			{"PATCH", item.Patch},
		}
		for _, mo := range candidates {
			if mo.op == nil {
				continue
			}
			capID := toUpperSnake(mo.op.OperationID)
			desc := mo.op.Summary
			if desc == "" {
				desc = mo.op.Description
			}
			e := capabilityEntry{
				ID:          capID,
				Category:    "api",
				Description: desc,
				Endpoint:    path,
				Method:      mo.method,
			}
			if mo.op.OperationID != capID {
				e.OperationID = mo.op.OperationID
			}
			entries = append(entries, e)
		}
	}

	entries = append(entries, actionCapabilities...)
	return entries
}

// toUpperSnake converte camelCase/PascalCase in UPPER_SNAKE_CASE.
// "GetPersons" → "GET_PERSONS", "InsertPerson" → "INSERT_PERSON"
func toUpperSnake(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	out := make([]rune, 0, len(runes)+4)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				out = append(out, '_')
			}
		}
		out = append(out, unicode.ToUpper(r))
	}
	return string(out)
}
