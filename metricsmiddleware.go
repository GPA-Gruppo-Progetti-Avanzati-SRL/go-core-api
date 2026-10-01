package coreapi

import (
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/slok/go-http-metrics/middleware"
)

type MetricsReporter struct {
	Middleware middleware.Middleware
}

func (m *MetricsReporter) MetricsHandler(ctx huma.Context, next func(huma.Context)) {

	mc := &MetricsContext{c: ctx, w: &middlewareResponseWriter{w: ctx.BodyWriter()}}

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
