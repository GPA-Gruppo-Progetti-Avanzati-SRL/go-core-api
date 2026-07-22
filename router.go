package apiservices

import (
	"context"
	"reflect"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/authorization"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-api/swagger"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
	"go.uber.org/fx"
)

// idempotentRegisterer wraps a prometheus.Registerer to silently ignore
// AlreadyRegisteredError. This allows NewRouter to be called multiple times
// (e.g., in tests) without panicking on duplicate metric registration.
type idempotentRegisterer struct{ prom.Registerer }

func (r idempotentRegisterer) Register(c prom.Collector) error {
	err := r.Registerer.Register(c)
	if _, ok := err.(prom.AlreadyRegisteredError); ok {
		return nil
	}
	return err
}

func (r idempotentRegisterer) MustRegister(cs ...prom.Collector) {
	for _, c := range cs {
		if err := r.Register(c); err != nil {
			panic(err)
		}
	}
}

type Router struct {
	Api huma.API
	Mux *chi.Mux
}
type Matcher struct {
	fx.In
	Authorizer coreauth.Authorizer `optional:"true"`
}

func newRouter(cm *chi.Mux, cfg *Config, matcher Matcher) *Router {
	r := &Router{
		Mux: cm,
	}
	var config huma.Config

	if cfg.DevelopMode && cfg.OpenApi != nil {
		config = huma.DefaultConfig(cfg.OpenApi.ApiName, cfg.OpenApi.ApiVersion)
		config.SchemasPath = ""
		config.CreateHooks = nil
		config.OpenAPI.Components.Schemas = ApiRegistry
		config.Components = &huma.Components{
			Schemas: ApiRegistry,
		}
		var serverList []*huma.Server
		for _, server := range cfg.OpenApi.Servers {
			serverList = append(serverList, &huma.Server{
				URL:         server.Url,
				Description: server.Description,
			})
		}
		r.Mux.Get("/openapi", swagger.Home)
		config.Servers = serverList
		config.DocsRenderer = huma.DocsRendererScalar
	} else {
		config = huma.DefaultConfig("", "")
		config.DocsPath = ""
		config.OpenAPIPath = ""
		// Senza OpenAPI esposta non serve lo SchemaLinkTransformer (aggiunto dai
		// CreateHooks di huma.DefaultConfig). Lasciarlo attivo lo farebbe girare a
		// ogni registrazione operazione e andrebbe in nil-pointer panic sugli schema
		// ($ref) che puntano ad ApiRegistry, non presente nel registry di questa
		// config. Lo disattiviamo come nel ramo develop-mode:true.
		config.CreateHooks = nil
	}

	// Nota: la configurazione Security non è più presente nel Config corrente;
	// lasciamo Components e Security invariati.

	reporter := &MetricsReporter{Middleware: middleware.New(middleware.Config{
		Service:  core.AppName,
		Recorder: prometheus.NewRecorder(prometheus.Config{Registry: idempotentRegisterer{prom.DefaultRegisterer}}),
	}),
	}
	r.Api = humachi.New(cm, config)

	// Endpoint di discovery: abilitati solo in develop-mode.
	// Registrati su chi direttamente: no auth, non appaiono nell'OpenAPI spec.
	if cfg.DevelopMode {
		cm.Get("/capabilities", capabilitiesHandler(r.Api))
		cm.Get("/capabilities.yaml", capabilitiesYAMLHandler(r.Api))
		cm.Get("/acl.mongo.js", capabilitiesMongoHandler(r.Api))
		cm.Get("/acl.sql", capabilitiesSQLHandler(r.Api))
	}

	r.Api.UseMiddleware(reporter.MetricsHandler)
	r.Api.UseMiddleware(tracingHandler)
	if cfg.Authorization != nil && cfg.Authorization.Enabled {
		// Inject authorizer in context for downstream middlewares/handlers
		if matcher.Authorizer != nil {
			r.Api.UseMiddleware(authorizerInjector(matcher.Authorizer))
			r.Api.UseMiddleware(authorization.AuthorizationHandler(cfg.Authorization))
			// Register standalone handlers (decoupled from Router)
			huma.Register(r.Api, authorization.TokenOperation, authorization.Token)
		} else {
			log.Fatal().Msg("No authorization operator  specified so i can't register the authorization middleware")
		}

	}
	r.Api.UseMiddleware(r.ValidatorHandler)
	configureError()
	return r
}

// authorizerInjector injects the provided Authorizer into the request context so that
// downstream middlewares and handlers can retrieve it without binding to Router.
func authorizerInjector(auth coreauth.Authorizer) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if auth != nil {
			ctx = huma.WithValue(ctx, "authorizer", auth)
		}
		next(ctx)
	}
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

func WithBusiness[D, Req, Resp any](dep D, fn func(context.Context, *Req, D) (*Resp, error)) func(context.Context, *Req) (*Resp, error) {
	return func(ctx context.Context, req *Req) (*Resp, error) {
		return fn(ctx, req, dep)
	}
}

// RegisterWithBusiness registra un'operazione Huma iniettando la dipendenza di
// business b nell'handler. Prende il *Router (invece della sola huma.API) così il
// sito di registrazione dipende da fx dal Router: questo ne forza il wiring (il
// costruttore è lazy e mode-gated) e ne garantisce la costruzione prima della
// registrazione. Prendere il Router — non un'API globale — permette anche di avere
// più router nello stesso processo.
func RegisterWithBusiness[B, Req, Resp any](
	r *Router,
	b B,
	op huma.Operation,
	fn func(context.Context, *Req, B) (*Resp, error),
) {
	huma.Register(r.Api, op, func(ctx context.Context, req *Req) (*Resp, error) {
		return fn(ctx, req, b)
	})
}
