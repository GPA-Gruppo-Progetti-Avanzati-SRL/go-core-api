package middleware

import (
	"net/http"
	"testing"
)

// writerOnly è un io.Writer che non è un http.ResponseWriter.
type writerOnly struct{ n int }

func (w *writerOnly) Write(p []byte) (int, error) { w.n += len(p); return len(p), nil }

// Il wrapper delle metriche avvolge un io.Writer qualunque: prima lo convertiva a
// http.ResponseWriter con una type assertion senza ok, e panicava.
func TestMiddlewareResponseWriter_WriterQualunque(t *testing.T) {
	w := &writerOnly{}
	mrw := &middlewareResponseWriter{w: w}
	_ = mrw.Header()
	mrw.WriteHeader(http.StatusTeapot)
	mrw.Flush()
	if _, err := mrw.Write([]byte("ciao")); err != nil || mrw.Length != 4 || w.n != 4 {
		t.Fatalf("Write: err=%v Length=%d scritti=%d", err, mrw.Length, w.n)
	}
}
