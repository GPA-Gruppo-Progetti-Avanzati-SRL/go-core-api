package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

// spanHeaders sono gli header di richiesta che finiscono sullo span, come
// `http.request.header.<nome>` (convenzione OTel). È una lista chiusa e non "tutti tranne alcuni"
// di proposito: prima ci finivano TUTTI, quindi Authorization, Cookie e i token applicativi
// arrivavano in chiaro al backend di tracing, con una retention e un controllo d'accesso che non
// sono quelli dell'API. Un header nuovo che porta un segreto non si vede in una deny-list.
var spanHeaders = []string{"Content-Type", "Accept", "User-Agent", "X-Request-Id", "X-Forwarded-For"}

// Tracing è il middleware huma che apre lo span OTel della richiesta.
func Tracing(ctx huma.Context, next func(huma.Context)) {
	// Tutti gli header servono all'estrazione del contesto di tracing (traceparent, baggage), ma
	// restano qui: sullo span vanno solo quelli di spanHeaders.
	headers := http.Header{}
	ctx.EachHeader(func(key, value string) { headers.Add(key, value) })
	sctx := otel.GetTextMapPropagator().Extract(ctx.Context(), propagation.HeaderCarrier(headers))

	sctx, span := otel.Tracer("huma").Start(sctx, ctx.Method())
	defer span.End()
	next(huma.WithContext(ctx, sctx))

	attributes := []attribute.KeyValue{
		attribute.String("http.request.method", ctx.Method()),
		attribute.String("http.route", ctx.Operation().Path),
		attribute.Int("http.response.status_code", ctx.Status()),
	}
	for _, h := range spanHeaders {
		if v := headers.Get(h); v != "" {
			attributes = append(attributes, attribute.String("http.request.header."+strings.ToLower(h), v))
		}
	}
	span.SetAttributes(attributes...)
	// Errore solo sui 5xx: un 4xx è un esito corretto del server a una richiesta sbagliata, e un 3xx
	// non è nemmeno un fallimento. Prima ogni status >= 300 marcava lo span in errore.
	if ctx.Status() >= 500 {
		span.SetStatus(codes.Error, "status "+strconv.Itoa(ctx.Status()))
	}
	span.SetName(fmt.Sprintf("%s %s", ctx.Method(), ctx.Operation().Path))
}
