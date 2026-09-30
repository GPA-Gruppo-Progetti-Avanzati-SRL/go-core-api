package coreapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProxyTarget(t *testing.T) {
	tests := []struct {
		raw, scheme, host string
		wantErr           bool
	}{
		{"legacy:8080", "http", "legacy:8080", false}, // la forma storica
		{"http://legacy:8080", "http", "legacy:8080", false},
		{"https://legacy:8443", "https", "legacy:8443", false},
		{"https://legacy:8443/", "https", "legacy:8443", false},
		{"", "", "", true},
		{"ftp://legacy", "", "", true},
		{"https://legacy/api", "", "", true}, // il path sarebbe ignorato in silenzio
		{"http://", "", "", true},
	}
	for _, tc := range tests {
		u, err := proxyTarget(tc.raw)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: atteso errore, ottenuto %v", tc.raw, u)
			}
			continue
		}
		if err != nil || u.Scheme != tc.scheme || u.Host != tc.host {
			t.Errorf("%q: %v %v, atteso %s://%s", tc.raw, u, err, tc.scheme, tc.host)
		}
	}
}

// Lo scheme era `http` fisso: un backend https riceveva una richiesta in chiaro sulla porta TLS.
func TestReverseProxy_BackendHTTPS(t *testing.T) {
	backend := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok "+r.URL.Path+" "+r.Header.Get("X-Extra"))
	}))
	defer backend.Close()

	// Il certificato del backend di test non è firmato da una CA di sistema: il test si fida di lui
	// sostituendo il DefaultTransport, da cui il proxy clona il proprio.
	orig := http.DefaultTransport
	http.DefaultTransport = backend.Client().Transport
	defer func() { http.DefaultTransport = orig }()

	h, err := NewReverseProxy(&ProxyConfig{
		MountPath: "/legacy", Url: backend.URL, Headers: []*Header{{Key: "X-Extra", Value: "v"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/legacy/x", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok /legacy/x v" {
		t.Fatalf("risposta %d %q", rec.Code, rec.Body.String())
	}
}

// Un backend che accetta la connessione e non risponde non tiene appesa la richiesta.
func TestReverseProxy_BackendCheNonRisponde(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer backend.Close()
	defer close(release)

	h, err := NewReverseProxy(&ProxyConfig{
		MountPath: "/legacy", Url: strings.TrimPrefix(backend.URL, "http://"),
		ResponseHeaderTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/legacy/x", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("codice %d, atteso 502", rec.Code)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("attesa %s: il timeout non è stato applicato", d)
	}
}

func TestReverseProxy_UrlNonValidoFermaLAvvio(t *testing.T) {
	if _, err := NewReverseProxy(&ProxyConfig{MountPath: "/x", Url: "ftp://legacy"}); err == nil {
		t.Fatal("atteso errore")
	}
}
