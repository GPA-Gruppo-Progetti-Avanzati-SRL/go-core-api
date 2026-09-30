package coreapi

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DefaultProxyResponseHeaderTimeout è l'attesa massima degli header di risposta del backend di un
// proxy (vedi ProxyConfig.ResponseHeaderTimeout).
const DefaultProxyResponseHeaderTimeout = 30 * time.Second

// NewReverseProxy costruisce il reverse proxy di una voce di `proxy:`. Ritorna errore se `url` non è
// una destinazione valida: l'app non parte, invece di rispondere 502 a ogni richiesta.
//
// Non attraversa i middleware huma — metriche, validazione, autorizzazione — perché è montato sul
// mux chi, fuori dalle operazioni: la catena che vale per il proxy è quella del mux (recover, limite
// al body) più lo span OTel aggiunto qui. Un backend esposto così è raggiungibile da chiunque
// raggiunga l'API: l'autorizzazione, se serve, è del backend.
func NewReverseProxy(pc *ProxyConfig) (http.Handler, error) {
	target, err := proxyTarget(pc.Url)
	if err != nil {
		return nil, fmt.Errorf("coreapi: proxy %q: %w", pc.MountPath, err)
	}

	// Il transport di default non ha un limite all'attesa della risposta: un backend che accetta la
	// connessione e non risponde tiene occupata la richiesta fino al write-timeout del server, e con
	// lei una connessione verso il backend. I timeout di dial e handshake del default restano.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = orDefault(pc.ResponseHeaderTimeout, DefaultProxyResponseHeaderTimeout)

	proxy := &httputil.ReverseProxy{
		// Rewrite e non Director (deprecata da Go 1.26): riceve la richiesta in ingresso (In) e
		// quella in uscita (Out) separate, quindi la riscrittura non può leggere per sbaglio uno
		// stato già modificato.
		Rewrite: func(pr *httputil.ProxyRequest) {
			// Scheme e host dalla destinazione configurata: prima lo scheme era `http` fisso, quindi
			// un backend https riceveva il TLS in chiaro. Il path originale resta quello della
			// richiesta in ingresso (lo gestisce Mount).
			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.Host = target.Host

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
		// Il transport strumentato propaga il contesto di tracing al backend.
		Transport: otelhttp.NewTransport(transport),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error().Err(err).Str("proxy", pc.MountPath).Str("target", target.Host).
				Msg("coreapi: backend del proxy irraggiungibile")
			http.Error(w, "Errore nel raggiungere il server remoto", http.StatusBadGateway)
		},
	}
	return otelhttp.NewHandler(proxy, "proxy "+pc.MountPath), nil
}

// proxyTarget interpreta `url`: con uno scheme (`https://legacy:8443`) vale quello, senza
// (`legacy:8080`, la forma storica) vale http.
func proxyTarget(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("url vuoto")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("url %q: %w", raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("url %q: scheme %q non supportato (http, https)", raw, u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url %q: host mancante", raw)
	}
	if u.Path != "" && u.Path != "/" {
		// Il path della richiesta resta quello in ingresso: un path qui sarebbe ignorato in silenzio.
		return nil, fmt.Errorf("url %q: il path non è supportato, la destinazione è solo scheme://host[:porta]", raw)
	}
	return u, nil
}
