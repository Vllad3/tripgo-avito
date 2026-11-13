package finish

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

type UseCase struct {
	trips   tripRepository
	history historyRepository
	tx      txManager
	log     *slog.Logger
	metrics metrics
	now     func() time.Time
}

func New(
	trips tripRepository,
	history historyRepository,
	tx txManager,
	log *slog.Logger,
	metrics metrics,
	now func() time.Time,
) *UseCase {
	return &UseCase{
		trips:   trips,
		history: history,
		tx:      tx,
		log:     log,
		metrics: metrics,
		now:     now,
	}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	var finished *trip.Trip

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		t, err := u.trips.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err := t.Finish(u.now().UTC()); err != nil {
			return err
		}
		if err := u.trips.Update(ctx, t); err != nil {
			return err
		}
		from := trip.StatusActive
		if err := u.history.Add(ctx, t.ID, &from, trip.StatusCompleted); err != nil {
			return err
		}
		finished = t
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	u.metrics.TripCompleted(ctx)
	u.log.InfoContext(ctx, "trip completed", slog.String("trip_id", finished.ID.String()))

	return finished, nil
}
