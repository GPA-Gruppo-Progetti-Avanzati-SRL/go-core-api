package apiservices

import (
	"encoding/json"
	"net/http"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

// capabilityEntry è il formato JSON esposto dall'endpoint GET /capabilities.
// Corrisponde al formato atteso dal discovery loader di app-fe.
// id:          ID UPPER_SNAKE locale (prefissato dal gateway con il proxy id)
// operationId: operationId originale Huma per Match() nel backend; omesso se uguale a id
// category:    "api" | "action_api"
type capabilityEntry struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Description string   `json:"description,omitempty"`
	OperationID string   `json:"operationId,omitempty"`
	Path        string   `json:"path,omitempty"`
	Methods     []string `json:"methods,omitempty"`
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

// capabilitiesHandler restituisce un http.HandlerFunc che serve GET /capabilities.
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
				Path:        path,
				Methods:     []string{mo.method},
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
