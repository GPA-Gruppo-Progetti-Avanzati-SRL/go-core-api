package api

import (
	"fmt"

	"github.com/danielgtaylor/huma/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/propagation"
	//"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"net/http"
	"strconv"
)

func TracingHandler(ctx huma.Context, next func(huma.Context)) {

	headers := http.Header{}
	attributes := make([]attribute.KeyValue, 0)
	ctx.EachHeader(func(key, value string) {
		headers.Add(key, value)
		attributes = append(attributes, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: attribute.StringValue(value),
		})
	})

	sctx := otel.GetTextMapPropagator().Extract(ctx.Context(), propagation.HeaderCarrier(headers))

	sctx, span := otel.Tracer("huma").Start(sctx, ctx.Method())

	defer span.End()
	hctx := huma.WithContext(ctx, sctx)
	next(hctx)
	if ctx.Status() >= 300 {
		span.SetStatus(codes.Error, "huma middleware failed with status code "+strconv.Itoa(ctx.Status()))
	}
	span.SetAttributes(attributes...)
	span.SetName(fmt.Sprintf("%s %s", ctx.Method(), ctx.Operation().Path))
}
