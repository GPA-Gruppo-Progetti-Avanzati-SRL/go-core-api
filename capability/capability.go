// Package capability deduce dal registry huma le capability che l'applicazione espone: una per
// operazione registrata, più le action_api dichiarate con RegisterActionCapability.
//
// È ciò che solo go-core-api può sapere. La serializzazione verso uno schema ACL appartiene a chi
// quello schema lo legge: il frontdoor OPEM (internal/opem, gli endpoint di develop-mode) e
// go-core-auth (apiauth, i seed /acl.coreauth.*).
package capability

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/danielgtaylor/huma/v2"
)

// CapabilityEntry è il formato JSON esposto dall'endpoint GET /capabilities.
//
// È esportata insieme a Capabilities e CapabilityID perché la generazione dei seed ACL non è
// tutta di questo modulo: qui sta ciò che solo qui si può sapere — quali capability l'applicazione
// espone, dedotte dal registry huma — mentre la serializzazione verso uno schema appartiene a chi
// quello schema lo legge.
// Corrisponde al formato atteso dal discovery loader di app-fe.
// id:          ID UPPER_SNAKE locale (prefissato dal gateway con il proxy id)
// operationId: operationId originale Huma per Match() nel backend; omesso se uguale a id
// category:    "api" | "action_api"
type CapabilityEntry struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Description string `json:"description,omitempty"`
	OperationID string `json:"operationId,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Method      string `json:"method,omitempty"`
}

// actionCapabilities raccoglie le capability action_api registrate dall'applicazione. Il mutex c'è
// perché la registrazione e la lettura (GET /capabilities) possono incontrarsi a runtime.
var (
	actionCapabilitiesMu sync.Mutex
	actionCapabilities   []CapabilityEntry
)

// RegisterActionCapability registra una capability action_api inclusa nella risposta
// di GET /capabilities. Chiamare durante l'inizializzazione dell'applicazione.
func RegisterActionCapability(id, description string) {
	actionCapabilitiesMu.Lock()
	defer actionCapabilitiesMu.Unlock()
	actionCapabilities = append(actionCapabilities, CapabilityEntry{
		ID:          id,
		Category:    "action_api",
		Description: description,
	})
}

// CapabilityID costruisce l'id strutturato di una capability: cap:<appID>:api:<id>. È la
// convenzione con cui le capability si nominano in tutti gli ACL della catena, quindi chi genera
// un seed deve usarla e non reinventarla.
func CapabilityID(appID, id string) string {
	return fmt.Sprintf("cap:%s:api:%s", appID, strings.ToLower(id))
}

// Capabilities elenca le capability che l'applicazione espone: una per operazione huma registrata,
// più le action_api dichiarate con RegisterActionCapability.
func Capabilities(api huma.API) []CapabilityEntry {
	openapi := api.OpenAPI()

	type methodOp struct {
		method string
		op     *huma.Operation
	}

	var entries []CapabilityEntry
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
			e := CapabilityEntry{
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

	actionCapabilitiesMu.Lock()
	entries = append(entries, actionCapabilities...)
	actionCapabilitiesMu.Unlock()
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
