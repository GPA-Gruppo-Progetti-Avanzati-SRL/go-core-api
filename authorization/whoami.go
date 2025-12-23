package authorization

import (
	"context"
	"net/http"

	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
)

var WhoamiResponses = map[string]*huma.Response{
	"200": {Ref: "", Description: "OK", Content: nil, Links: nil, Extensions: nil},
}

type whoAmIResponse struct {
	Body *whoamiBodyResponse
}

type whoamiBodyResponse struct {
	User         string   `json:"user"`
	Roles        []string `json:"roles"`
	Capabilities []string `json:"capabilities"`
}

// whoamiRequest consente di passare opzionalmente l'AppId via header
type whoamiRequest struct {
	AppID string `header:"AppId"`
}

var WhoamiOperation = huma.Operation{
	OperationID:   "whoami",
	Method:        http.MethodGet,
	Path:          "/api/whoami",
	Summary:       "Informazioni sull'utente corrente e i permessi derivati dai ruoli",
	Tags:          []string{"system"},
	DefaultStatus: http.StatusOK,
	Responses:     WhoamiResponses,
}

// Whoami restituisce informazioni sull'utente e capacità derivate dai ruoli, leggendo dal context
func Whoami(ctx context.Context, i *whoamiRequest) (*whoAmIResponse, error) {
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
	if v := ctx.Value("authorizer"); v != nil {
		if auth, ok := v.(coreauth.Authorizer); ok && auth != nil {
			if i != nil && i.AppID != "" {
				caps = auth.GetCapabilities(roles, i.AppID)
			} else {
				caps = auth.GetCapabilities(roles)
			}
		}
	}

	body := &whoamiBodyResponse{User: user, Roles: roles, Capabilities: caps}
	return &whoAmIResponse{body}, nil

}
