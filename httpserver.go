package coreapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

func newService(lc fx.Lifecycle, sh fx.Shutdowner, cfg *Config) *chi.Mux {
	mux := chi.NewRouter()
	server := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	for _, pc := range cfg.Proxy {
		mux.Mount(pc.MountPath, NewReverseProxy(pc))
	}
	if cfg.Idle == 0 {
		cfg.Idle = 30 * time.Second
	}
	srv := &http.Server{
		Addr:        server,
		Handler:     mux,
		IdleTimeout: cfg.Idle,
		// A 0 net/http applica DefaultMaxHeaderValueCount: nessun default da
		// duplicare qui.
		MaxHeaderValueCount: cfg.MaxHeaderValueCount,
	}

	lc.Append(fx.Hook{
		// /metrics e /health si montano all'avvio e non qui: chi rifiuta un Use dopo la prima
		// rotta, e i middleware del Router si aggiungono dopo la costruzione del mux. L'hook è
		// appeso prima di quello del server, quindi gira prima del listen.
		OnStart: func(context.Context) error {
			mux.Handle("/metrics", promhttp.Handler())
			mux.Handle("/health", core.HealthHandler)
			return nil
		},
	})
	// Listen in OnStart, Shutdown col context dell'hook, e un accept loop che muore fa uscire il
	// processo: prima qui si loggava e basta, contando su una probe su /health servita dallo
	// stesso server morto.
	core.ServeOnLifecycle(lc, sh, srv, "api")
	return mux
}
