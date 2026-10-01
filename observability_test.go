package coreapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type probeOut struct{ Body struct{ Ok bool } }

// Sullo span vanno solo gli header della lista chiusa, e un 4xx non è un errore del server: prima ci
// finivano tutti gli header — Authorization compreso — e ogni status >= 300 marcava lo span in errore.
func TestTracing_HeaderERiusciti(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	mux := chi.NewRouter()
	r := newRouter(mux, &Config{})
	huma.Register(r.Api, huma.Operation{OperationID: "probe", Method: http.MethodGet, Path: "/probe/{code}"},
		func(_ context.Context, in *struct {
			Code int `path:"code"`
		}) (*probeOut, error) {
			if in.Code != 200 {
				return nil, huma.NewError(in.Code, "no")
			}
			return &probeOut{}, nil
		})

	for _, tc := range []struct {
		path    string
		isError bool
	}{{"/probe/200", false}, {"/probe/404", false}, {"/probe/500", true}} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("Authorization", "Bearer segreto")
		req.Header.Set("Cookie", "session=segreto")
		req.Header.Set("User-Agent", "probe")
		mux.ServeHTTP(httptest.NewRecorder(), req)

		spans := rec.Ended()
		s := spans[len(spans)-1]
		var ua bool
		for _, a := range s.Attributes() {
			if strings.Contains(a.Value.String(), "segreto") {
				t.Fatalf("%s: un segreto è finito sullo span: %s=%s", tc.path, a.Key, a.Value.String())
			}
			if a.Key == "http.request.header.user-agent" && a.Value.AsString() == "probe" {
				ua = true
			}
		}
		if !ua {
			t.Errorf("%s: User-Agent assente dallo span", tc.path)
		}
		if got := s.Status().Code == codes.Error; got != tc.isError {
			t.Errorf("%s: span in errore = %v, atteso %v", tc.path, got, tc.isError)
		}
	}
}
