package authorization

import (
	"context"
	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

var MenuResponses = map[string]*huma.Response{
	"200": {Ref: "", Description: "OK", Content: nil, Links: nil, Extensions: nil},
}

type menuResponse struct {
	Body []*coreauth.MenuNode
}

// menuRequest consente di passare opzionalmente l'AppId via header
type menuRequest struct {
	AppID string `header:"AppId"`
}

var MenuOperation = huma.Operation{
	OperationID:   "menu",
	Method:        http.MethodGet,
	Path:          "/api/menu",
	Summary:       "Albero menu",
	Tags:          []string{"system"},
	DefaultStatus: http.StatusOK,
	Responses:     MenuResponses,
}

// Menu restituisce l'albero dei menu per i ruoli correnti, leggendo dal context
func Menu(ctx context.Context, i *menuRequest) (*menuResponse, error) {
	// Estrae i ruoli dal context impostato dal middleware di autorizzazione
	var roles []string
	if v := ctx.Value("roles"); v != nil {
		if rr, ok := v.([]string); ok {
			roles = rr
		}
	}

	// Delego all'authorizer la costruzione del menu per i ruoli correnti
	// Si assume che l'authorizer applichi le regole e ritorni l'albero di menu.
	var tree []*coreauth.MenuNode
	if v := ctx.Value("authorizer"); v != nil {
		if auth, ok := v.(coreauth.Authorizer); ok && auth != nil {
			if i != nil && i.AppID != "" {
				tree = auth.GetMenu(roles, i.AppID)
			} else {
				tree = auth.GetMenu(roles)
			}
		}
	}

	return &menuResponse{Body: tree}, nil
}
