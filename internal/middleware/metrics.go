package middleware

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	core "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"github.com/danielgtaylor/huma/v2"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

// Metrics ritorna il middleware huma delle metriche HTTP (go-http-metrics su Prometheus), etichettate
// con il nome dell'app.
func Metrics() func(huma.Context, func(huma.Context)) {
	reporter := &metricsReporter{Middleware: middleware.New(middleware.Config{
		Service:  core.AppName,
		Recorder: prometheus.NewRecorder(prometheus.Config{Registry: idempotentRegisterer{prom.DefaultRegisterer}}),
	})}
	return reporter.MetricsHandler
}

// idempotentRegisterer wraps a prometheus.Registerer to silently ignore
// AlreadyRegisteredError. This allows the router to be built multiple times
// (e.g., in tests) without panicking on duplicate metric registration.
type idempotentRegisterer struct{ prom.Registerer }

func (r idempotentRegisterer) Register(c prom.Collector) error {
	err := r.Registerer.Register(c)
	if _, ok := err.(prom.AlreadyRegisteredError); ok {
		return nil
	}
	return err
}

func (r idempotentRegisterer) MustRegister(cs ...prom.Collector) {
	for _, c := range cs {
		if err := r.Register(c); err != nil {
			panic(err)
		}
	}
}

type metricsReporter struct {
	Middleware middleware.Middleware
}

func (m *metricsReporter) MetricsHandler(ctx huma.Context, next func(huma.Context)) {

	mc := &metricsContext{c: ctx, w: &middlewareResponseWriter{w: ctx.BodyWriter()}}

	m.Middleware.Measure("", mc, func() {
		next(mc)
	})

}

type metricsContext struct {
	c huma.Context
	w *middlewareResponseWriter
}

func (r *metricsContext) TLS() *tls.ConnectionState {
	return r.c.TLS()

}

func (r *metricsContext) Version() huma.ProtoVersion {
	return r.c.Version()
}

func (r *metricsContext) Operation() *huma.Operation {
	return r.c.Operation()
}

func (r *metricsContext) Host() string {
	return r.c.Host()
}

func (r *metricsContext) RemoteAddr() string {
	return r.c.RemoteAddr()
}

func (r *metricsContext) URL() url.URL {
	return r.c.URL()
}

func (r *metricsContext) Param(name string) string {
	return r.c.Param(name)
}

func (r *metricsContext) Query(name string) string {
	return r.c.Query(name)
}

func (r *metricsContext) Header(name string) string {
	return r.c.Header(name)
}

func (r *metricsContext) EachHeader(cb func(name string, value string)) {
	r.c.EachHeader(cb)
}

func (r *metricsContext) BodyReader() io.Reader {
	return r.c.BodyReader()
}

func (r *metricsContext) GetMultipartForm() (*multipart.Form, error) {
	return r.c.GetMultipartForm()
}

func (r *metricsContext) SetReadDeadline(time time.Time) error {
	return r.c.SetReadDeadline(time)
}

func (r *metricsContext) SetStatus(code int) {
	r.c.SetStatus(code)
}

func (r *metricsContext) Status() int {
	return r.c.Status()
}

func (r *metricsContext) SetHeader(name, value string) {
	r.c.SetHeader(name, value)
}

func (r *metricsContext) AppendHeader(name, value string) {
	r.c.AppendHeader(name, value)
}

func (r *metricsContext) Method() string {
	return r.c.Method()
}

func (r *metricsContext) URLPath() string {
	return r.c.Operation().Path
}

func (r *metricsContext) StatusCode() int { return r.c.Status() }

func (r *metricsContext) BytesWritten() int64 {
	return r.w.Length
}
func (r *metricsContext) BodyWriter() io.Writer {
	return r.w
}
func (r *metricsContext) Context() context.Context {
	return r.c.Context()
}

// middlewareResponseWriter conta i byte scritti nel body, per la metrica della dimensione della
// risposta. Avvolge il BodyWriter di huma, che è un io.Writer: prima lo si convertiva a
// http.ResponseWriter con una type assertion senza ok, e un adapter huma con un writer di altro
// tipo faceva panicare ogni richiesta dentro il middleware delle metriche.
type middlewareResponseWriter struct {
	w      io.Writer
	Length int64
}

func (crw *middlewareResponseWriter) Header() http.Header {
	if rw, ok := crw.w.(http.ResponseWriter); ok {
		return rw.Header()
	}
	return http.Header{}
}

func (crw *middlewareResponseWriter) WriteHeader(status int) {
	if rw, ok := crw.w.(http.ResponseWriter); ok {
		rw.WriteHeader(status)
	}
}

func (crw *middlewareResponseWriter) Write(p []byte) (int, error) {
	n, err := crw.w.Write(p)
	crw.Length += int64(n)
	return n, err
}

// Flush e Unwrap tengono raggiungibili le capacità del writer avvolto: senza, una risposta in
// streaming (SSE) che chiede il flush al BodyWriter lo perde dietro questo wrapper.
func (crw *middlewareResponseWriter) Flush() {
	if f, ok := crw.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (crw *middlewareResponseWriter) Unwrap() io.Writer { return crw.w }
