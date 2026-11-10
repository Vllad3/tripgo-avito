package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(log *slog.Logger, route func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tmpl := route(r)
			start := time.Now()

			observed(next, w, r, func(status int) {
				level := slog.LevelInfo
				if status >= http.StatusInternalServerError {
					level = slog.LevelError
				}
				log.LogAttrs(r.Context(), level, "request handled",
					slog.String("method", r.Method),
					slog.String("route", tmpl),
					slog.Int("status", status),
					slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				)
			})
		})
	}
}
