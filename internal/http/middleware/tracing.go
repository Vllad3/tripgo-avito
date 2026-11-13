package middleware

import (
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "trip-service/http"

func Tracing(tp trace.TracerProvider, route func(*http.Request) string) func(http.Handler) http.Handler {
	tracer := tp.Tracer(tracerName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			tmpl := route(r)
			ctx, span := tracer.Start(ctx, r.Method+" "+tmpl,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", tmpl),
					attribute.String("url.path", r.URL.Path),
				),
			)
			defer span.End()

			observed(next, w, r.WithContext(ctx), func(status int) {
				span.SetAttributes(attribute.Int("http.response.status_code", status))

				if status >= http.StatusInternalServerError {
					span.SetStatus(codes.Error, http.StatusText(status))
					span.SetAttributes(attribute.String("error.type", strconv.Itoa(status)))
				}
			})
		})
	}
}
