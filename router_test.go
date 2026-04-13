package apiservices

import (
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
		ApiName:         "Test API",
		ApiVersion:      "1.0.0",
		OpenApiDisabled: true,
	}

	_ = NewRouter(mux, cfg, Matcher{})

	// Check /openapi (the custom route)
	req, _ := http.NewRequest("GET", "/openapi", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)

	// Check Huma default docs path /docs
	req, _ = http.NewRequest("GET", "/docs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)

	// Check Huma default openapi path /openapi.json
	req, _ = http.NewRequest("GET", "/openapi.json", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestOpenApiEnabled(t *testing.T) {
	mux := chi.NewRouter()
	cfg := &Config{
		ApiName:         "Test API",
		ApiVersion:      "1.0.0",
		OpenApiDisabled: false,
	}

	router := NewRouter(mux, cfg, Matcher{})

	// Huma might register routes after NewRouter if not using humachi.New directly which we are.
	// Actually humachi.New registers them immediately.

	// Check /openapi (the custom route)
	req, _ := http.NewRequest("GET", "/openapi", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check Huma default docs path /docs (default in huma.DefaultConfig)
	req, _ = http.NewRequest("GET", "/docs", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Register a dummy route to ensure OpenAPI generation works
	huma.Get(router.Api, "/test", func(ctx huma.Context, input *struct{}) (*struct{ Body string }, error) {
		return &struct{ Body string }{Body: "ok"}, nil
	})

	// Check Huma default openapi path /openapi.json
	req, _ = http.NewRequest("GET", "/openapi.json", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
