package coreapi

import (
	"github.com/rs/zerolog/log"
	"net/http"
	"net/http/httputil"
)

func NewReverseProxy(pc *ProxyConfig) http.Handler {
	// Configura il reverse proxy
	proxy := &httputil.ReverseProxy{
		// Rewrite e non Director (deprecata da Go 1.26): riceve la richiesta in ingresso (In) e
		// quella in uscita (Out) separate, quindi la riscrittura non può leggere per sbaglio uno
		// stato già modificato.
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Imposta l'URL di destinazione. Il path originale resta quello della richiesta in
			// ingresso (lo gestisce Mount).
			pr.Out.URL.Scheme = "http" // o "https" se necessario
			pr.Out.URL.Host = pc.Url

			// Imposta l'Host header
			pr.Out.Host = pc.Url

			// Director accodava X-Forwarded-For da sé, Rewrite no: va chiesto. La riga sulla
			// catena in ingresso è necessaria perché SetXForwarded, da solo, la SOSTITUISCE con
			// il solo peer diretto — dietro un frontdoor si perderebbe l'IP del client vero.
			// Rispetto a Director si aggiungono X-Forwarded-Host e X-Forwarded-Proto.
			pr.Out.Header["X-Forwarded-For"] = pr.In.Header["X-Forwarded-For"]
			pr.SetXForwarded()

			for _, v := range pc.Headers {
				pr.Out.Header.Set(v.Key, v.Value)
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error().Err(err).Msgf("Errore nel proxy: %v", err)
			http.Error(w, "Errore nel raggiungere il server remoto", http.StatusBadGateway)
		},
	}

	// Restituisci il proxy come handler
	return proxy
}
