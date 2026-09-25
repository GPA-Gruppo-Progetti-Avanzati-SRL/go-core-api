package coreapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
)

// Seed delle capability per go-core-auth.
//
// Sono endpoint distinti da /acl.sql e /acl.mongo.js, che restano al vocabolario del frontdoor
// OPEM (`opem_acl_cap_def`, `_et: "cap-def"`). I due mondi hanno schemi diversi e destinatari
// diversi: fonderli significherebbe generare per l'uno un seed che l'altro non sa leggere, che è
// esattamente la situazione da cui questi endpoint nascono.
//
// Nessuno dei due script assegna capability a un ruolo: creano le capability e il gruppo che le
// raccoglie. Assegnare il gruppo a un ruolo resta un atto esplicito di chi governa l'ACL — ed è la
// ragione per cui eseguire il seed non concede permessi a nessuno.

// capabilitiesCoreAuthSQLHandler serve GET /acl.coreauth.sql → upsert sulle tabelle acl_* lette da
// go-core-auth/sqlsource.
func capabilitiesCoreAuthSQLHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toCoreAuthSQL(buildEntries(api))))
	}
}

// capabilitiesCoreAuthMongoHandler serve GET /acl.coreauth.js → upsert sulla collection acl letta
// da go-core-auth/mongosource.
func capabilitiesCoreAuthMongoHandler(api huma.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(toCoreAuthMongo(buildEntries(api))))
	}
}

// coreAuthGroupID è il gruppo che raccoglie tutte le capability dell'applicazione.
func coreAuthGroupID(appID string) string { return fmt.Sprintf("grp:%s:ALL", appID) }

// toCoreAuthSQL serializza le capability su acl_capability, acl_capability_group e
// acl_capability_group_item. Sintassi PostgreSQL / SQLite.
func toCoreAuthSQL(entries []capabilityEntry) string {
	appID := core.AppName
	var sb strings.Builder

	sb.WriteString("-- Capability di " + appID + " per go-core-auth/sqlsource.\n")
	sb.WriteString("-- Generato da GET /acl.coreauth.sql — idempotente, sintassi PostgreSQL / SQLite.\n")
	sb.WriteString("-- Non assegna il gruppo ad alcun ruolo: quella resta una decisione esplicita.\n\n")

	for _, e := range entries {
		fmt.Fprintf(&sb,
			"INSERT INTO acl_capability (id, category, description, operation_id, api_path, api_methods,\n"+
				"                            endpoint, icon, ord, menu, app_id)\n"+
				"VALUES (%s, %s, %s, %s, %s, %s, '', '', 0, FALSE, %s)\n"+
				"ON CONFLICT (id) DO UPDATE SET\n"+
				"    category = EXCLUDED.category, description = EXCLUDED.description,\n"+
				"    operation_id = EXCLUDED.operation_id, api_path = EXCLUDED.api_path,\n"+
				"    api_methods = EXCLUDED.api_methods, app_id = EXCLUDED.app_id;\n\n",
			sqlStr(capID(appID, e.ID)), sqlStr(e.Category), sqlStr(descriptionOf(e)),
			sqlStr(operationIDOf(e)), sqlStr(e.Endpoint), sqlStr(e.Method), sqlStr(appID),
		)
	}

	groupID := coreAuthGroupID(appID)
	fmt.Fprintf(&sb,
		"INSERT INTO acl_capability_group (id, description)\n"+
			"VALUES (%s, %s)\n"+
			"ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description;\n\n",
		sqlStr(groupID), sqlStr("Tutte le capability di "+appID),
	)
	for _, e := range entries {
		fmt.Fprintf(&sb,
			"INSERT INTO acl_capability_group_item (group_id, capability_id) VALUES (%s, %s)\n"+
				"ON CONFLICT (group_id, capability_id) DO NOTHING;\n",
			sqlStr(groupID), sqlStr(capID(appID, e.ID)),
		)
	}
	return sb.String()
}

// coreAuthCapDoc è il documento CAPABILITY della collection acl. I campi dell'api stanno in un
// sottodocumento, come il modello di mongosource: è la categoria a dire quale sottodocumento
// riguarda la capability.
type coreAuthCapDoc struct {
	ID          string             `json:"_id"`
	ET          string             `json:"_et"`
	Category    string             `json:"category"`
	Description string             `json:"description,omitempty"`
	AppID       string             `json:"appId,omitempty"`
	API         *coreAuthCapAPIDoc `json:"api,omitempty"`
}

type coreAuthCapAPIDoc struct {
	OperationID string   `json:"operationid,omitempty"`
	Path        string   `json:"path,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}

// toCoreAuthMongo serializza le capability come replaceOne upsert sulla collection acl.
func toCoreAuthMongo(entries []capabilityEntry) string {
	appID := core.AppName
	var sb strings.Builder

	sb.WriteString("// Capability di " + appID + " per go-core-auth/mongosource.\n")
	sb.WriteString("// Generato da GET /acl.coreauth.js — idempotente.\n")
	sb.WriteString("// Non assegna il gruppo ad alcun ruolo: quella resta una decisione esplicita.\n\n")
	sb.WriteString("const COLLECTION = \"acl\";\n\n")

	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		id := capID(appID, e.ID)
		ids = append(ids, id)

		doc := coreAuthCapDoc{
			ID: id, ET: "CAPABILITY", Category: e.Category,
			Description: descriptionOf(e), AppID: appID,
		}
		// Il sottodocumento api riguarda le sole capability che autorizzano una rotta: su una
		// action_api sarebbe un campo che nessuno legge, e il validator della collection lo rifiuta.
		if e.Category == "api" {
			doc.API = &coreAuthCapAPIDoc{OperationID: operationIDOf(e), Path: e.Endpoint}
			if e.Method != "" {
				doc.API.Methods = []string{e.Method}
			}
		}
		writeUpsert(&sb, id, doc)
	}

	groupID := coreAuthGroupID(appID)
	writeUpsert(&sb, groupID, struct {
		ID           string   `json:"_id"`
		ET           string   `json:"_et"`
		Description  string   `json:"description"`
		Capabilities []string `json:"capabilities"`
	}{groupID, "CAPABILITYGROUP", "Tutte le capability di " + appID, ids})

	return sb.String()
}

func writeUpsert(sb *strings.Builder, id string, doc any) {
	raw, _ := json.MarshalIndent(doc, "    ", "    ")
	fmt.Fprintf(sb,
		"db.getCollection(COLLECTION).replaceOne(\n    { _id: %s },\n    %s,\n    { upsert: true }\n)\n\n",
		jsonStr(id), string(raw),
	)
}

// descriptionOf: una capability senza descrizione resta comunque riconoscibile dal suo id.
func descriptionOf(e capabilityEntry) string {
	if e.Description != "" {
		return e.Description
	}
	return e.ID
}

// operationIDOf restituisce l'operationId di huma. buildEntries lo omette quando coincide con l'id
// della capability, ma nel seed va scritto comunque: è il campo con cui un ACL basato
// sull'operationId trova la capability.
func operationIDOf(e capabilityEntry) string {
	if e.OperationID != "" {
		return e.OperationID
	}
	return e.ID
}
