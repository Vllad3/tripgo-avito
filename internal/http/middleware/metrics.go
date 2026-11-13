package middleware

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(
	reqDuration metric.Float64Histogram,
	activeRequests metric.Int64UpDownCounter,
	route func(*http.Request) string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tmpl := route(r)
			base := metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", tmpl),
			)

			activeRequests.Add(r.Context(), 1, base)
			start := time.Now()

			observed(next, w, r, func(status int) {
				activeRequests.Add(r.Context(), -1, base)
				reqDuration.Record(
					r.Context(), time.Since(start).Seconds(),
					metric.WithAttributes(
						attribute.String("http.request.method", r.Method),
						attribute.String("http.route", tmpl),
						attribute.Int("http.response.status_code", status),
					),
				)
			})
		})
	}
}
