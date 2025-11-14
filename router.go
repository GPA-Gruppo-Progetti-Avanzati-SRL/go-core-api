package apiservices

import (
	"reflect"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/authorization"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/swagger"
	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

type Router struct {
	Api         huma.API
	Mux         *chi.Mux
	roleMatcher authorization.RoleMatcher
}

func NewRouter(cm *chi.Mux, cfg *Config, matcher authorization.RoleMatcher) *Router {
	r := &Router{
		Mux:         cm,
		roleMatcher: matcher,
	}

	config := huma.DefaultConfig(cfg.ApiName, cfg.ApiVersion)
	config.SchemasPath = ""
	config.CreateHooks = nil
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

	r.Mux.Get("/openapi", swagger.Home)

	// Nota: la configurazione Security non è più presente nel Config corrente;
	// lasciamo Components e Security invariati.
	config.Servers = serverList

	reporter := &MetricsReporter{Middleware: middleware.New(middleware.Config{
		Service:  core.AppName,
		Recorder: prometheus.NewRecorder(prometheus.Config{}),
	}),
	}
	r.Api = humachi.New(cm, config)
	r.Api.UseMiddleware(reporter.MetricsHandler)
	r.Api.UseMiddleware(TracingHandler)
	if cfg.Authorization != nil && cfg.Authorization.Enabled {
		r.Api.UseMiddleware(r.AuthorizationHandler(cfg.Authorization))
		huma.Register(r.Api, authorization.WhoamiOperation, authorization.Whoami)
	}
	r.Api.UseMiddleware(r.ValidatorHandler)

	ConfigureError()
	return r
}

var ApiRegistry = huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)

var DefaultResponses = map[string]*huma.Response{
	"400": {Ref: "", Description: "BadRequest/Validation Error", Content: ErrorContent, Links: nil, Extensions: nil},
	"404": {Ref: "", Description: "Not Found", Content: ErrorContent, Links: nil, Extensions: nil},
	"408": {Ref: "", Description: "Request Timeout", Content: nil, Links: nil, Extensions: nil},
	"422": {Ref: "", Description: "KO Applicativo", Content: ErrorContent, Links: nil, Extensions: nil},
	"500": {Ref: "", Description: "Internal Server Error", Content: ErrorContent, Links: nil, Extensions: nil},
}

func SerializeSchema(input interface{}) *huma.Schema {
	return ApiRegistry.Schema(reflect.TypeOf(input), true, "")

}
