package coreapi

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/httpx"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/observability"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/fx"
)

func newService(lc fx.Lifecycle, sh fx.Shutdowner, cfg *Config) (*chi.Mux, error) {
	mux := chi.NewRouter()
	server := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	// In testa, prima di qualsiasi rotta (chi non accetta un Use dopo): valgono per tutto ciò che
	// il server serve, proxy compresi. Il recover è il più esterno, così copre anche il limite.
	mux.Use(recoverer, limitBody(orDefault(cfg.MaxBodyBytes, DefaultMaxBodyBytes)))

	for _, pc := range cfg.Proxy {
		proxy, err := NewReverseProxy(pc)
		if err != nil {
			return nil, err
		}
		mux.Mount(pc.MountPath, proxy)
	}
	srv := &http.Server{
		Addr:              server,
		Handler:           mux,
		IdleTimeout:       orDefault(cfg.Idle, DefaultIdleTimeout),
		ReadHeaderTimeout: orDefault(cfg.ReadHeaderTimeout, DefaultReadHeaderTimeout),
		ReadTimeout:       orDefault(cfg.ReadTimeout, DefaultReadTimeout),
		WriteTimeout:      orDefault(cfg.WriteTimeout, DefaultWriteTimeout),
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
			mux.Handle("/health", observability.HealthHandler)
			return nil
		},
	})
	// Listen in OnStart, Shutdown col context dell'hook, e un accept loop che muore fa uscire il
	// processo: prima qui si loggava e basta, contando su una probe su /health servita dallo
	// stesso server morto.
	httpx.ServeOnLifecycle(lc, sh, srv, "api")
	return mux, nil
}
