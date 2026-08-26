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

	_ = newRouter(mux, cfg, Matcher{})

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

	router := newRouter(mux, cfg, Matcher{})

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

// --- default responses -------------------------------------------------------

type defRespABody struct {
	AField string `json:"aField"`
}
type defRespBBody struct {
	BField int `json:"bField"`
}
type defRespAOutput struct {
	Body defRespABody
}
type defRespBOutput struct {
	Body defRespBBody
}

func TestWithDefaultResponsesMergesAndCopies(t *testing.T) {
	op := withDefaultResponses(huma.Operation{
		OperationID: "Custom",
		Responses:   map[string]*huma.Response{"404": {Description: "Persona non trovata"}},
	})

	// i default ci sono tutti, ma la chiave dichiarata dall'operazione vince
	for _, code := range []string{"400", "408", "422", "500"} {
		assert.Equal(t, DefaultResponses[code].Description, op.Responses[code].Description, code)
	}
	assert.Equal(t, "Persona non trovata", op.Responses["404"].Description)

	// la mappa e i Response sono copie: scriverci non tocca il globale
	op.Responses["500"].Description = "mutato"
	op.Responses["599"] = &huma.Response{}
	assert.Equal(t, "Internal Server Error", DefaultResponses["500"].Description)
	assert.NotContains(t, DefaultResponses, "599")
}

// Le operazioni non condividono i *huma.Response: huma riempie la response di
// successo scrivendo dentro op.Responses, quindi due operazioni registrate una
// dopo l'altra devono avere ognuna il proprio schema (in passato il maps.Copy
// per-operazione nell'app serviva a garantirlo).
func TestRegisterWithBusinessIsolatesResponses(t *testing.T) {
	mux := chi.NewRouter()
	cfg := &Config{
		DevelopMode: true,
		OpenApi:     &OpenApiConfig{ApiName: "Test API", ApiVersion: "1.0.0"},
	}
	router := newRouter(mux, cfg, Matcher{})

	RegisterWithBusiness(router, struct{}{},
		huma.Operation{OperationID: "GetA", Method: http.MethodGet, Path: "/a", DefaultStatus: http.StatusOK},
		func(ctx context.Context, in *struct{}, b struct{}) (*defRespAOutput, error) { return nil, nil })
	RegisterWithBusiness(router, struct{}{},
		huma.Operation{OperationID: "GetB", Method: http.MethodGet, Path: "/b", DefaultStatus: http.StatusOK,
			Responses: map[string]*huma.Response{"404": {Description: "Custom Not Found"}}},
		func(ctx context.Context, in *struct{}, b struct{}) (*defRespBOutput, error) { return nil, nil })

	spec := router.Api.OpenAPI()
	a := spec.Paths["/a"].Get.Responses
	b := spec.Paths["/b"].Get.Responses

	// errori standard documentati su entrambe, senza init()+maps.Copy nell'app
	for _, code := range []string{"400", "404", "408", "422", "500"} {
		assert.NotNil(t, a[code], "GetA %s", code)
		assert.NotNil(t, b[code], "GetB %s", code)
	}
	assert.Equal(t, "Not Found", a["404"].Description)
	assert.Equal(t, "Custom Not Found", b["404"].Description)
	assert.Same(t, ErrorContent[ApplicationJson].Schema, a["400"].Content[ApplicationJson].Schema)

	// ognuna con il proprio schema di successo, sintetizzato da huma da Output.Body
	refA := a["200"].Content[ApplicationJson].Schema.Ref
	refB := b["200"].Content[ApplicationJson].Schema.Ref
	assert.NotEmpty(t, refA)
	assert.NotEqual(t, refA, refB)

	// il globale non è stato toccato: nessun "200", nessun header
	assert.NotContains(t, DefaultResponses, "200")
	assert.Len(t, DefaultResponses, 5)
	for code, r := range DefaultResponses {
		assert.Nil(t, r.Headers, code)
	}
}
