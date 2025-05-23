package apiservices

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"
	"net"
	"net/http"
)

func NewService(lc fx.Lifecycle, cfg *Config) *chi.Mux {
	mux := chi.NewRouter()
	server := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	mux.Handle("/metrics", promhttp.Handler())

	for _, pc := range cfg.Proxy {
		mux.Mount(pc.MountPath, NewReverseProxy(pc))
	}

	srv := &http.Server{Addr: server, Handler: mux}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			log.Info().Msgf("Starting HTTP server at %s", srv.Addr)
			go srv.Serve(ln)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return mux
}
