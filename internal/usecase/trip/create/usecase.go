package create

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

type Input struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	Start    trip.Point
	End      trip.Point
	Price    int64
}

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	log     *slog.Logger
	metrics metrics
	now     func() time.Time
	newID   func() uuid.UUID
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	log *slog.Logger,
	metrics metrics,
	now func() time.Time,
	newID func() uuid.UUID,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		log:     log,
		metrics: metrics,
		now:     now,
		newID:   newID,
	}
}

func (u *UseCase) Execute(ctx context.Context, in Input) (*trip.Trip, error) {
	t, err := trip.New(u.newID(), in.UserID, in.DriverID, in.Start, in.End, in.Price, u.now().UTC())
	if err != nil {
		return nil, err
	}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.trips.Save(ctx, t); err != nil {
			return err
		}
		return u.history.Add(ctx, t.ID, nil, trip.StatusActive)
	})
	if err != nil {
		return nil, fmt.Errorf("create trip: %w", err)
	}

	u.metrics.TripCreated(ctx)
	u.log.InfoContext(ctx, "trip created",
		slog.String("trip_id", t.ID.String()),
		slog.String("user_id", t.UserID.String()),
		slog.String("driver_id", t.DriverID.String()),
	)

	return t, nil
}
