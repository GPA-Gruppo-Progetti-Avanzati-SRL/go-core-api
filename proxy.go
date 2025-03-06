package apiservices

import (
	"github.com/rs/zerolog/log"
	"net/http"
	"net/http/httputil"
)

func NewReverseProxy(pc *ProxyConfig) http.Handler {
	// Configura il reverse proxy
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			// Imposta l'URL di destinazione
			req.URL.Scheme = "http" // o "https" se necessario
			req.URL.Host = pc.Url
			// Mantieni il path originale della richiesta
			// (viene automaticamente gestito da Mount)

			// Imposta l'Host header
			req.Host = pc.Url

			for _, v := range pc.Headers {
				req.Header.Set(v.Key, v.Value)
			}

			// Aggiungi header per tracciare il proxy (opzionale)
			//req.Header.Set("X-Proxy-Origin", "chi-reverse-proxy")
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error().Err(err).Msgf("Errore nel proxy: %v", err)
			http.Error(w, "Errore nel raggiungere il server remoto", http.StatusBadGateway)
		},
	}

	// Restituisci il proxy come handler
	return proxy
}
