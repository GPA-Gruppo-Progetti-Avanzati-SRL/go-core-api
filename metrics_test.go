package coreapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/fx/fxtest"
)

// Il /metrics dell'API serve il registry di default, e una metrica creata con otel.Meter deve
// comparirci senza che l'app installi niente: senza MeterProvider sarebbe no-op, ed è così che le
// metriche del pool di tpm-mongo-common restavano invisibili. Il server non viene avviato: basta
// il costruttore, che è dove il provider viene installato.
func TestMetrics_StrumentiOtelEsposti(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	if _, err := newService(lc, nil, &Config{Host: "127.0.0.1", Port: 0}); err != nil {
		t.Fatalf("newService: %v", err)
	}

	counter, err := otel.Meter("go-core-api-test").Int64UpDownCounter("coreapi.test.otel.visibile")
	if err != nil {
		t.Fatalf("strumento otel: %v", err)
	}
	counter.Add(context.Background(), 3)

	rr := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("/metrics: status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "coreapi_test_otel_visibile") {
		t.Fatalf("la metrica otel non compare su /metrics:\n%s", rr.Body.String())
	}
}
