package apiservices

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"reflect"
)

type Router struct {
	Api huma.API
}

func NewRouter(cm *chi.Mux, reporter *MetricsReporter, cfg *Config) *Router {
	r := &Router{}

	config := huma.DefaultConfig(cfg.ApiName, cfg.ApiVersion)
	config.SchemasPath = ""
	config.OpenAPI.Components.Schemas = ApiRegistry
	config.Components = &huma.Components{
		Schemas: ApiRegistry,
	}
	var serverList []*huma.Server
	for _, server := range cfg.Servers {
		serverList = append(serverList, &huma.Server{
			URL:         server.Url,
			Description: server.Description,
		})
	}

	config.Servers = serverList
	r.Api = humachi.New(cm, config)
	r.Api.UseMiddleware(reporter.MetricsHandler)
	r.Api.UseMiddleware(TracingHandler)
	r.Api.UseMiddleware(r.ValidatorHandler)
	ConfigureError()
	return r
}

var ApiRegistry = huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)

var DefaultResponses = map[string]*huma.Response{
	"400": {Ref: "", Description: "BadRequest/Validation Error", Content: ErrorContent, Links: nil, Extensions: nil},
	"408": {Ref: "", Description: "Request Timeout", Content: nil, Links: nil, Extensions: nil},
	"422": {Ref: "", Description: "KO Applicativo", Content: ErrorContent, Links: nil, Extensions: nil},
	"500": {Ref: "", Description: "Internal Server Error", Content: ErrorContent, Links: nil, Extensions: nil},
}

func SerializeSchema(input interface{}) *huma.Schema {
	return ApiRegistry.Schema(reflect.TypeOf(input), true, "")

}
