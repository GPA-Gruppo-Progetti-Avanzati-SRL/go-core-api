package authorization

import (
	"net/http"
	"strings"

	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
	"github.com/rs/zerolog/log"
)

// AuthorizationHandler restituisce un middleware che verifica la presenza
// di un ruolo/funzione nell'header (iniettato da un proxy) corrispondente
// all'endpoint invocato. L'identificatore richiesto è per default l'OperationID
// dell'endpoint; in alternativa può usare metodo+path.
func AuthorizationHandler(cfg *Config) func(huma.Context, func(huma.Context)) {
	// Imposta default sicuri
	rolesHeader := "X-Roles"
	contextHeader := "X-Context"
	userHeader := "X-User"
	delimiter := ","

	guestPaths := map[string]struct{}{
		"/openapi": {},
		"/metrics": {},
		"/health":  {},
	}
	if cfg != nil {
		if cfg.RolesHeader != "" {
			rolesHeader = cfg.RolesHeader
		}
		if cfg.ContextHeader != "" {
			contextHeader = cfg.ContextHeader
		}
		if cfg.UserHeader != "" {
			userHeader = cfg.UserHeader
		}
		if cfg.Delimiter != "" {
			delimiter = cfg.Delimiter
		}
		if len(cfg.GuestPaths) > 0 {
			guestPaths = map[string]struct{}{}
			for _, p := range cfg.GuestPaths {
				guestPaths[p] = struct{}{}
			}
		}
	}

	return func(ctx huma.Context, next func(huma.Context)) {
		// Esclusioni iniziali
		if strings.EqualFold(ctx.Method(), http.MethodOptions) {
			next(ctx)
			return
		}

		path := ctx.URL().Path
		if _, ok := guestPaths[path]; ok {
			next(ctx)
			return
		}

		rolesStr := ctx.Header(rolesHeader)
		log.Trace().Msgf("Roles header %s", rolesStr)

		if rolesStr == "" {
			deny(ctx)
			return
		}

		allRoles := parseRoles(rolesStr, delimiter)
		user := strings.TrimSpace(ctx.Header(userHeader))
		contextId := strings.TrimSpace(ctx.Header(contextHeader))
		log.Trace().Msgf("User header %s, Context header %s", user, contextId)

		// Recupera l'authorizer eventualmente iniettato da AuthorizerInjector
		var authorizer coreauth.Authorizer
		if v := ctx.Context().Value("authorizer"); v != nil {
			if a, ok := v.(coreauth.Authorizer); ok {
				authorizer = a
			}
		}

		// Filtraggio per contesto: se presente X-Context, riduce i ruoli a quelli validi
		contextRoles := allRoles
		if contextId != "" && authorizer != nil {
			contextRoles = authorizer.FilterRolesByContext(allRoles, contextId)
			if len(contextRoles) == 0 {
				denyContext(ctx)
				return
			}
		}

		// Arricchisce il context Huma con i valori richiesti
		ctx = huma.WithValue(ctx, "user", user)
		ctx = huma.WithValue(ctx, "contextId", contextId)
		ctx = huma.WithValue(ctx, "allRoles", allRoles)  // tutti i ruoli (per GetApps nel token)
		ctx = huma.WithValue(ctx, "roles", contextRoles) // ruoli del contesto corrente

		// Se non è stato iniettato alcun matcher, consenti (policy attuale) ma avvisa
		if authorizer == nil {
			log.Warn().Msg("No role matcher specified so i pass")
			next(ctx)
			return
		}

		if !authorizer.MatchRequest(contextRoles, path, ctx.Method()) {
			deny(ctx)
			return
		}
		next(ctx)
	}
}

func parseRoles(v, delimiter string) []string {
	parts := strings.Split(v, delimiter)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func deny(ctx huma.Context) {
	ctx.SetStatus(http.StatusForbidden)
	ctx.SetHeader("Content-Type", "application/json")
	_, _ = ctx.BodyWriter().Write([]byte(`{"error":"forbidden","message":"missing required role"}`))
}

func denyContext(ctx huma.Context) {
	ctx.SetStatus(http.StatusForbidden)
	ctx.SetHeader("Content-Type", "application/json")
	_, _ = ctx.BodyWriter().Write([]byte(`{"error":"forbidden","message":"context not authorized"}`))
}
