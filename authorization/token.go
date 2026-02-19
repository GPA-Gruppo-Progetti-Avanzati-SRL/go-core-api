package authorization

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
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
	Roles        []string         `json:"roles"`
	Capabilities []string         `json:"capabilities"`
	Apps         []*coreauth.App  `json:"apps"`
	Paths        []*coreauth.Path `json:"paths"`
}

// whoamiRequest consente di passare opzionalmente l'AppId via header
type tokenRequest struct {
	AppID string `header:"AppId"  required:"true"`
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
func Token(ctx context.Context, i *tokenRequest) (*RawStringOutput, error) {
	// Estrae user e roles dal context impostato dal middleware di autorizzazione
	var user string
	if v := ctx.Value("user"); v != nil {
		if s, ok := v.(string); ok {
			user = s
		}
	}
	var roles []string
	if v := ctx.Value("roles"); v != nil {
		if rr, ok := v.([]string); ok {
			roles = rr
		}
	}

	// Recupera l'authorizer dal context, se presente
	var caps []string
	var apps []*coreauth.App
	var paths []*coreauth.Path
	if v := ctx.Value("authorizer"); v != nil {
		if auth, ok := v.(coreauth.Authorizer); ok && auth != nil {

			apps = auth.GetApps(roles)
			paths = auth.GetPaths(roles, i.AppID)
			caps = auth.GetCapabilities(roles, i.AppID)

		}
	}

	body := &tokenBodyResponse{User: user, Roles: roles, Capabilities: caps, Apps: apps, Paths: paths}
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
