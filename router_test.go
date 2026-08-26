package coreapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestAliasOnlyRegistration registra operazioni costruite ESCLUSIVAMENTE con i tipi
// riesportati da coreapi (nessun identificatore huma), che è il percorso che le app
// devono poter seguire per non dichiarare huma come require diretto.
func TestAliasOnlyRegistration(t *testing.T) {
	mux := chi.NewRouter()
	router := newRouter(mux, &Config{
		DevelopMode: true,
		OpenApi:     &OpenApiConfig{ApiName: "Alias API", ApiVersion: "1.0.0"},
	}, Matcher{})

	type body struct {
		Name string `json:"name"`
	}
	type out struct {
		Body any
	}

	// Caso "Output.Body any": schema dichiarato a mano con Response + MediaType +
	// SerializeSchema, tutti presi da coreapi.
	RegisterWithBusiness(router, struct{}{}, Operation{
		OperationID:   "AliasOnly",
		Method:        http.MethodGet,
		Path:          "/alias-only",
		Summary:       "Operazione senza import di huma",
		DefaultStatus: http.StatusOK,
		Responses: map[string]*Response{
			"200": {Description: "OK", Content: map[string]*MediaType{
				ApplicationJson: {Schema: SerializeSchema(body{})},
			}},
		},
	}, func(ctx context.Context, req *struct{}, _ struct{}) (*out, error) {
		return &out{Body: body{Name: "ok"}}, nil
	})

	// Multipart: huma riconosce la richiesta SOLO se il tipo concreto di RawBody è
	// MultipartFormFiles[T]. È la prova che l'alias preserva il match per reflection.
	type uploadIn struct {
		RawBody MultipartFormFiles[struct {
			File FormFile `form:"file" contentType:"application/octet-stream" required:"false"`
			Meta string   `form:"metadata" required:"true"`
		}]
	}
	RegisterWithBusiness(router, struct{}{}, Operation{
		OperationID:   "AliasUpload",
		Method:        http.MethodPost,
		Path:          "/alias-upload",
		Summary:       "Upload senza import di huma",
		DefaultStatus: http.StatusOK,
	}, func(ctx context.Context, req *uploadIn, _ struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})

	req, _ := http.NewRequest("GET", "/openapi.json", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
			RequestBody *struct {
				Content map[string]any `json:"content"`
			} `json:"requestBody"`
		} `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &spec))

	get := spec.Paths["/alias-only"]["get"]
	assert.Equal(t, "AliasOnly", get.OperationID)

	// Gli errori standard restano mergiati anche passando per i tipi alias.
	post := spec.Paths["/alias-upload"]["post"]
	assert.Equal(t, "AliasUpload", post.OperationID)
	require.NotNil(t, post.RequestBody, "il RawBody MultipartFormFiles deve produrre un requestBody")
	assert.Contains(t, post.RequestBody.Content, "multipart/form-data",
		"huma deve riconoscere il multipart dal tipo concreto preservato dall'alias")
}
