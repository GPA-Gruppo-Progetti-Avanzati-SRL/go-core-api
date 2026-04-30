package apiservices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestOpenApiDisabled(t *testing.T) {
	mux := chi.NewRouter()
	cfg := &Config{
		OpenApi: nil,
	}

	_ = NewRouter(mux, cfg, Matcher{})

	req, _ := http.NewRequest("GET", "/openapi", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)

	req, _ = http.NewRequest("GET", "/docs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)

	req, _ = http.NewRequest("GET", "/openapi.json", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestOpenApiEnabled(t *testing.T) {
	mux := chi.NewRouter()
	cfg := &Config{
		DevelopMode: true,
		OpenApi: &OpenApiConfig{
			ApiName:    "Test API",
			ApiVersion: "1.0.0",
		},
	}

	router := NewRouter(mux, cfg, Matcher{})

	req, _ := http.NewRequest("GET", "/openapi", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	req, _ = http.NewRequest("GET", "/docs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	huma.Get(router.Api, "/test", func(ctx context.Context, input *struct{}) (*struct{ Body string }, error) {
		return &struct{ Body string }{Body: "ok"}, nil
	})

	req, _ = http.NewRequest("GET", "/openapi.json", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
