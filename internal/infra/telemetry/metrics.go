package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type HTTPMetrics struct {
	RequestDuration metric.Float64Histogram
	ActiveRequests  metric.Int64UpDownCounter
}

type TripMetrics struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewHTTPMetrics(mp metric.MeterProvider) (*HTTPMetrics, error) {
	meter := mp.Meter(serviceName)

	reqDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of inbound HTTP requests."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create http.server.request.duration: %w", err)
	}

	active, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of inbound HTTP requests in flight."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create http.server.active_requests: %w", err)
	}

	return &HTTPMetrics{RequestDuration: reqDuration, ActiveRequests: active}, nil
}

func NewTripMetrics(mp metric.MeterProvider) (*TripMetrics, error) {
	meter := mp.Meter(serviceName)

	created, err := meter.Int64Counter(
		"tripgo.trips.created",
		metric.WithDescription("Number of trips created successfully."),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.created: %w", err)
	}

	completed, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metric.WithDescription("Number of trips completed successfully."),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.completed: %w", err)
	}

	return &TripMetrics{created: created, completed: completed}, nil
}

func (m *TripMetrics) TripCreated(ctx context.Context) {
	m.created.Add(ctx, 1)
}

func (m *TripMetrics) TripCompleted(ctx context.Context) {
	m.completed.Add(ctx, 1)
}
