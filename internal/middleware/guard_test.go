package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Il resto del comportamento di Recoverer e LimitBody è provato in coreapi (guard_test.go), sul mux
// come lo monta newService.

func TestRecoverer_ErrAbortHandlerRilanciato(t *testing.T) {
	h := Recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if p := recover(); p != http.ErrAbortHandler {
			t.Fatalf("http.ErrAbortHandler va rilanciato, recover = %v", p)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
