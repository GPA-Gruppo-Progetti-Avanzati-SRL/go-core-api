package api

import (
	"context"
	"crypto/tls"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

type MetricsReporter struct {
	Middleware middleware.Middleware
	sm         *core.ServerMetrics
}

func NewMetricsReporter(sm *core.ServerMetrics) *MetricsReporter {

	return &MetricsReporter{Middleware: middleware.New(middleware.Config{
		Service:  core.AppName,
		Recorder: prometheus.NewRecorder(prometheus.Config{}),
	}),
		sm: sm}
}

func (m *MetricsReporter) MetricsHandler(ctx huma.Context, next func(huma.Context)) {

	mc := &MetricsContext{c: ctx, w: &middlewareResponseWriter{rw: (ctx.BodyWriter()).(http.ResponseWriter)}}

	m.Middleware.Measure("", mc, func() {
		next(mc)
	})

}

type MetricsContext struct {
	c huma.Context
	w *middlewareResponseWriter
}

func (r *MetricsContext) TLS() *tls.ConnectionState {
	return r.c.TLS()

}

func (r *MetricsContext) Version() huma.ProtoVersion {
	return r.c.Version()
}

func (r *MetricsContext) Operation() *huma.Operation {
	return r.c.Operation()
}

func (r *MetricsContext) Host() string {
	return r.c.Host()
}

func (r *MetricsContext) RemoteAddr() string {
	return r.c.RemoteAddr()
}

func (r *MetricsContext) URL() url.URL {
	return r.c.URL()
}

func (r *MetricsContext) Param(name string) string {
	return r.c.Param(name)
}

func (r *MetricsContext) Query(name string) string {
	return r.c.Query(name)
}

func (r *MetricsContext) Header(name string) string {
	return r.c.Header(name)
}

func (r *MetricsContext) EachHeader(cb func(name string, value string)) {
	r.c.EachHeader(cb)
}

func (r *MetricsContext) BodyReader() io.Reader {
	return r.c.BodyReader()
}

func (r *MetricsContext) GetMultipartForm() (*multipart.Form, error) {
	return r.c.GetMultipartForm()
}

func (r *MetricsContext) SetReadDeadline(time time.Time) error {
	return r.c.SetReadDeadline(time)
}

func (r *MetricsContext) SetStatus(code int) {
	r.c.SetStatus(code)
}

func (r *MetricsContext) Status() int {
	return r.c.Status()
}

func (r *MetricsContext) SetHeader(name, value string) {
	r.c.SetHeader(name, value)
}

func (r *MetricsContext) AppendHeader(name, value string) {
	r.c.AppendHeader(name, value)
}

func (r *MetricsContext) Method() string {
	return r.c.Method()
}

func (r *MetricsContext) URLPath() string {
	return r.c.Operation().Path
}

func (r *MetricsContext) StatusCode() int { return r.c.Status() }

func (r *MetricsContext) BytesWritten() int64 {
	return r.w.Length
}
func (r *MetricsContext) BodyWriter() io.Writer {
	return r.w
}
func (r *MetricsContext) Context() context.Context {
	return r.c.Context()
}

type middlewareResponseWriter struct {
	rw     http.ResponseWriter
	Length int64
}

func (crw *middlewareResponseWriter) Header() http.Header {
	return crw.rw.Header()
}

func (crw *middlewareResponseWriter) WriteHeader(status int) {
	crw.rw.WriteHeader(status)
}

func (crw *middlewareResponseWriter) Write(p []byte) (int, error) {

	n, err := crw.rw.Write(p)
	crw.Length += int64(n)
	return n, err
}
