package authorization

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
)

var TokenResponses = map[string]*huma.Response{
	"200": {Description: "OK", Content: map[string]*huma.MediaType{"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}}}},
}

type RawStringOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

func (r *RawStringOutput) MarshalJSON() ([]byte, error) {
	return r.Body, nil
}

type tokenBodyResponse struct {
	User         string           `json:"user"`
	Context      string           `json:"context,omitempty"`
	Roles        []string         `json:"roles"`
	Capabilities []string         `json:"capabilities"`
	Apps         []*coreauth.App  `json:"apps"`
	Paths        []*coreauth.Path `json:"paths"`
}

// tokenRequest consente di passare opzionalmente l'AppId via header
type tokenRequest struct {
	AppID string `header:"AppId" required:"true"`
}

var TokenOperation = huma.Operation{
	OperationID:   "Token",
	Method:        http.MethodGet,
	Path:          "/api/token",
	Summary:       "Informazioni sull'utente corrente e i permessi derivati dai ruoli",
	Tags:          []string{"system"},
	DefaultStatus: http.StatusOK,
	Responses:     TokenResponses,
}

// Token restituisce informazioni sull'utente e capacità derivate dai ruoli, leggendo dal context.
// Il risultato è cifrato in formato hex usando l'AppID come chiave.
// Apps usa allRoles per visione globale delle app navigabili (incluse le istanze multicontext).
// Capabilities e Paths usano i ruoli filtrati per contesto (contextRoles).
func Token(ctx context.Context, i *tokenRequest) (*RawStringOutput, error) {
	var user string
	if v := ctx.Value("user"); v != nil {
		if s, ok := v.(string); ok {
			user = s
		}
	}

	// contextRoles: ruoli filtrati per il contesto corrente (per Match, GetPaths, GetCapabilities)
	var contextRoles []string
	if v := ctx.Value("roles"); v != nil {
		if rr, ok := v.([]string); ok {
			contextRoles = rr
		}
	}

	// allRoles: tutti i ruoli dell'utente (per GetApps — visione globale delle app navigabili)
	allRoles := contextRoles // fallback: se allRoles non è nel context usa contextRoles
	if v := ctx.Value("allRoles"); v != nil {
		if rr, ok := v.([]string); ok {
			allRoles = rr
		}
	}

	var contextId string
	if v := ctx.Value("contextId"); v != nil {
		if s, ok := v.(string); ok {
			contextId = s
		}
	}

	var caps []string
	var apps []*coreauth.App
	var paths []*coreauth.Path
	if v := ctx.Value("authorizer"); v != nil {
		if auth, ok := v.(coreauth.Authorizer); ok && auth != nil {
			apps = auth.GetApps(allRoles, contextId)           // scoped al contesto corrente
			paths = auth.GetPaths(contextRoles, i.AppID)       // scoped al contesto corrente
			caps = auth.GetCapabilities(contextRoles, i.AppID) // scoped al contesto corrente
		}
	}

	body := &tokenBodyResponse{
		User:         user,
		Context:      contextId,
		Roles:        contextRoles,
		Capabilities: caps,
		Apps:         apps,
		Paths:        paths,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	cipherText, err := core.Encrypt(b, i.AppID)
	if err != nil {
		return nil, huma.Error500InternalServerError("encryption error", err)
	}

	return &RawStringOutput{Body: []byte(hex.EncodeToString(cipherText)), ContentType: "text/plain"}, nil
}
