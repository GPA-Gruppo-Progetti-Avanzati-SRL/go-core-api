package coreapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

type guardItem struct {
	Name string `json:"name" validate:"required,min=2"`
}

type guardStructIn struct {
	Body guardItem
}

type guardArrayIn struct {
	Body []guardItem
}

type guardOut struct {
	Body struct {
		Ok bool `json:"ok"`
	}
}

// guardServer è il mux come lo monta newService — recoverer e limite al body in testa — con il
// Router e le sue middleware huma sopra.
func guardServer(t *testing.T, maxBody int64) http.Handler {
	t.Helper()
	mux := chi.NewRouter()
	mux.Use(recoverer, limitBody(maxBody))
	router := newRouter(mux, &Config{OpenApi: &OpenApiConfig{ApiName: "guard", ApiVersion: "1"}})
	ok := func() (*guardOut, error) { o := &guardOut{}; o.Body.Ok = true; return o, nil }
	RegisterWithBusiness(router, struct{}{},
		huma.Operation{OperationID: "PostStruct", Method: http.MethodPost, Path: "/struct"},
		func(ctx context.Context, in *guardStructIn, _ struct{}) (*guardOut, error) { return ok() })
	RegisterWithBusiness(router, struct{}{},
		huma.Operation{OperationID: "PostArray", Method: http.MethodPost, Path: "/array"},
		func(ctx context.Context, in *guardArrayIn, _ struct{}) (*guardOut, error) { return ok() })
	RegisterWithBusiness(router, struct{}{},
		huma.Operation{OperationID: "GetPanic", Method: http.MethodGet, Path: "/panic"},
		func(ctx context.Context, in *struct{}, _ struct{}) (*guardOut, error) { panic("boom") })
	return mux
}

func do(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, DefaultError) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var de DefaultError
	_ = json.Unmarshal(rr.Body.Bytes(), &de)
	return rr, de
}

// Un body con un array al top-level: lo schema non ha un Ref, e TypeFromRef("") panicava.
func TestValidator_ArrayTopLevelNonPanica(t *testing.T) {
	h := guardServer(t, 1<<20)
	if rr, _ := do(t, h, http.MethodPost, "/array", `[{"name":"ab"}]`); rr.Code != http.StatusOK {
		t.Fatalf("status %d, body %s", rr.Code, rr.Body)
	}
}

func TestValidator_TagValidateAncoraApplicati(t *testing.T) {
	h := guardServer(t, 1<<20)
	if rr, _ := do(t, h, http.MethodPost, "/struct", `{"name":"ab"}`); rr.Code != http.StatusOK {
		t.Fatalf("body valido: status %d, body %s", rr.Code, rr.Body)
	}
	if rr, _ := do(t, h, http.MethodPost, "/struct", `{"name":"a"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("validate:min=2 violato: status %d, body %s", rr.Code, rr.Body)
	}
}

// Un body oltre il limite si ferma con un 413, non arriva all'handler troncato.
func TestLimitBody_413(t *testing.T) {
	h := guardServer(t, 32)
	rr, de := do(t, h, http.MethodPost, "/struct", `{"name":"`+strings.Repeat("x", 100)+`"}`)
	if rr.Code != http.StatusRequestEntityTooLarge || de.Code != CodeBodyTooLarge {
		t.Fatalf("status %d code %q, attesi 413 %s; body %s", rr.Code, de.Code, CodeBodyTooLarge, rr.Body)
	}
}

// Un handler che panica dà un 500 in forma DefaultError, invece di una connessione chiusa.
func TestRecoverer_PanicDiventa500(t *testing.T) {
	h := guardServer(t, 1<<20)
	rr, de := do(t, h, http.MethodGet, "/panic", "")
	if rr.Code != http.StatusInternalServerError || de.Code != CodePanic || de.Ambit != Ambit {
		t.Fatalf("status %d, errore %+v", rr.Code, de)
	}
}

func TestRecoverer_ErrAbortHandlerRilanciato(t *testing.T) {
	h := recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Fatalf("http.ErrAbortHandler va rilanciato, recover = %v", p)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestOrDefault(t *testing.T) {
	if orDefault[int64](0, 5) != 5 || orDefault[int64](-1, 5) != 0 || orDefault[int64](7, 5) != 7 {
		t.Fatal("orDefault: 0 = default, negativo = disattivato, positivo = valore")
	}
}
