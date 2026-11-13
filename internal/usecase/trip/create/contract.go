package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type (
	tripRepository interface {
		Save(ctx context.Context, t *trip.Trip) error
	}

	historyRepository interface {
		Add(ctx context.Context, tripID uuid.UUID, from *trip.Status, to trip.Status) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		TripCreated(ctx context.Context)
	}
)
