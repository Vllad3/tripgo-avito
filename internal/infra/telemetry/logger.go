package telemetry

import (
	"fmt"
	"io"
	"log/slog"
)

const serviceName = "trip-service"

func NewLogger(w io.Writer, level string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL %q: %w", level, err)
	}

	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler).With(slog.String("service", serviceName)), nil
}
