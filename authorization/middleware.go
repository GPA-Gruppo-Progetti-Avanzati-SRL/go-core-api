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
	delimiter := ","
	userHeader := "X-User"

	guestPaths := map[string]struct{}{
		"/openapi": {},
		"/metrics": {},
		"/health":  {},
	}
	if cfg != nil {
		if cfg.RolesHeader != "" {
			rolesHeader = cfg.RolesHeader
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
		if _, ok := guestPaths[ctx.URL().Path]; ok {
			next(ctx)
			return
		}

		op := ctx.Operation()
		var required string

		required = op.OperationID
		log.Trace().Msgf("Operation id %s", required)

		// Se non c'è modo di determinare l'identificatore, lascia passare
		if required == "" {
			log.Debug().Msg("Authorization: nessun identificatore endpoint, bypass")
			next(ctx)
			return
		}

		rolesStr := ctx.Header(rolesHeader)
		if rolesStr == "" {
			// Niente ruoli: forbidden
			deny(ctx, rolesHeader, required)
			return
		}
		// Parsing header in lista di ruoli usando il delimitatore configurato
		roles := parseRoles(rolesStr, delimiter)
		// Propaga nel context anche l'utente (se presente) e la lista ruoli
		user := strings.TrimSpace(ctx.Header(userHeader))
		// Arricchisce il context Huma con i valori richiesti
		ctx = huma.WithValue(ctx, "user", user)
		ctx = huma.WithValue(ctx, "roles", roles)

		// Recupera l'authorizer eventualmente iniettato da AuthorizerInjector
		var authorizer coreauth.Authorizer
		if v := ctx.Context().Value("authorizer"); v != nil {
			if a, ok := v.(coreauth.Authorizer); ok {
				authorizer = a
			}
		}

		// Se non è stato iniettato alcun matcher, consenti (policy attuale) ma avvisa
		if authorizer == nil {
			log.Warn().Msg("No role matcher specified so i pass")
			next(ctx)
			return
		}
		if !authorizer.Match(roles, required) {
			deny(ctx, rolesHeader, required)
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

func deny(ctx huma.Context, rolesHeader, required string) {
	ctx.SetStatus(http.StatusForbidden)
	ctx.SetHeader("Content-Type", "application/json")
	body := `{"error":"forbidden","message":"missing required role","required":"` + required + `","header":"` + rolesHeader + `"}`
	_, _ = ctx.BodyWriter().Write([]byte(body))
}
