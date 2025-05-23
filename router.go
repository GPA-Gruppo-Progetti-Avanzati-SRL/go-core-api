package apiservices

import (
	"reflect"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/swagger"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

type Router struct {
	Api huma.API
	Mux *chi.Mux
}

func NewRouter(cm *chi.Mux, cfg *Config) *Router {
	r := &Router{
		Mux: cm,
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

	var security []map[string][]string
	config.Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
	for _, sc := range cfg.Security {

		config.Components.SecuritySchemes[sc.Key] = &huma.SecurityScheme{
			Type:         sc.Type,
			Scheme:       sc.Scheme,
			BearerFormat: sc.BearerFormat,
			Name:         sc.Name,
			Description:  sc.Description,
			In:           sc.In,
		}

		security = append(security, map[string][]string{
			sc.Key: {},
		})
	}
	config.Security = append(config.Security, security...)
	config.Servers = serverList

	reporter := &MetricsReporter{Middleware: middleware.New(middleware.Config{
		Service:  core.AppName,
		Recorder: prometheus.NewRecorder(prometheus.Config{}),
	}),
	}
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
	"404": {Ref: "", Description: "Not Found", Content: ErrorContent, Links: nil, Extensions: nil},
	"408": {Ref: "", Description: "Request Timeout", Content: nil, Links: nil, Extensions: nil},
	"422": {Ref: "", Description: "KO Applicativo", Content: ErrorContent, Links: nil, Extensions: nil},
	"500": {Ref: "", Description: "Internal Server Error", Content: ErrorContent, Links: nil, Extensions: nil},
}

func SerializeSchema(input interface{}) *huma.Schema {
	return ApiRegistry.Schema(reflect.TypeOf(input), true, "")

}
