package coreapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
)

func newService(lc fx.Lifecycle, cfg *Config) *chi.Mux {
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

		OnStart: func(ctx context.Context) error {
			mux.Handle("/metrics", promhttp.Handler())
			mux.Handle("/health", core.HealthHandler)
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info().Msgf("Starting HTTP server at %s", srv.Addr)
			go func() {
				// Serve ritorna ErrServerClosed a ogni arresto ordinato (OnStop → Shutdown):
				// quello è l'esito atteso. Qualsiasi altro errore significa che l'accept loop
				// è morto e l'API non risponde più, mentre il processo resta su: senza questa
				// riga non ci sarebbe nulla a dirlo. Il recovery lo fa l'orchestratore, che vede
				// fallire la probe su /health — servita da questo stesso server.
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Error().Err(err).Msgf("HTTP server terminato su %s: l'API non risponde più", srv.Addr)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return mux
}
