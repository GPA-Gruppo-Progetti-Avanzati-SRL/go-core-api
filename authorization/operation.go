package authorization

import (
	"context"

	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

var WhoamiResponses = map[string]*huma.Response{
	"200": {Ref: "", Description: "OK", Content: nil, Links: nil, Extensions: nil},
}

type whoAmIInput struct {
	// Nessun binding da header: i valori arrivano dal context impostato dal middleware
}
type whoAmIResponse struct {
	Body *whoamiBodyResponse
}

type whoamiBodyResponse struct {
	User              string   `json:"user"`
	Roles             []string `json:"roles"`
	AllowedOperations []string `json:"allowedOperations"`
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

func Whoami(ctx context.Context, in *whoAmIInput) (*whoAmIResponse, error) {
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

	/*if matcher == nil {
		return nil, core.BusinessErrorWithCodeAndMessage("ERR_ROLER", "Roler not specified")
	}
	*/
	allowed := getAllowed(roles)
	body := &whoamiBodyResponse{User: user, Roles: roles, AllowedOperations: allowed}
	return &whoAmIResponse{body}, nil
}

func getAllowed(roles []string) []string {

	allowed := make([]string, 0)
	/*oapi := r.Api.OpenAPI()
	if oapi != nil && oapi.Paths != nil {
		for _, pi := range oapi.Paths {
			if pi.Get != nil && pi.Get.OperationID != "" && r.roleMatcher.Match(roles, pi.Get.OperationID) {
				allowed = append(allowed, pi.Get.OperationID)
			}
			if pi.Put != nil && pi.Put.OperationID != "" && r.roleMatcher.Match(roles, pi.Put.OperationID) {
				allowed = append(allowed, pi.Put.OperationID)
			}
			if pi.Post != nil && pi.Post.OperationID != "" && r.roleMatcher.Match(roles, pi.Post.OperationID) {
				allowed = append(allowed, pi.Post.OperationID)
			}
			if pi.Delete != nil && pi.Delete.OperationID != "" && r.roleMatcher.Match(roles, pi.Delete.OperationID) {
				allowed = append(allowed, pi.Delete.OperationID)
			}
			if pi.Patch != nil && pi.Patch.OperationID != "" && r.roleMatcher.Match(roles, pi.Patch.OperationID) {
				allowed = append(allowed, pi.Patch.OperationID)
			}
			if pi.Head != nil && pi.Head.OperationID != "" && r.roleMatcher.Match(roles, pi.Head.OperationID) {
				allowed = append(allowed, pi.Head.OperationID)
			}
		}
	}*/
	return allowed
}
