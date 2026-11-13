package trip

import (
	"time"

	"github.com/google/uuid"

	domain "github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
}

type tripRow struct {
	id         uuid.UUID
	userID     uuid.UUID
	driverID   uuid.UUID
	startLat   float64
	startLon   float64
	endLat     float64
	endLon     float64
	price      int64
	status     string
	startedAt  time.Time
	finishedAt *time.Time
}

func (r *tripRow) dest() []any {
	return []any{
		&r.id, &r.userID, &r.driverID,
		&r.startLat, &r.startLon, &r.endLat, &r.endLon,
		&r.price, &r.status, &r.startedAt, &r.finishedAt,
	}
}

func (r *tripRow) toDomain() *domain.Trip {
	return &domain.Trip{
		ID:         r.id,
		UserID:     r.userID,
		DriverID:   r.driverID,
		Start:      domain.Point{Latitude: r.startLat, Longitude: r.startLon},
		End:        domain.Point{Latitude: r.endLat, Longitude: r.endLon},
		Price:      r.price,
		Status:     domain.Status(r.status),
		StartedAt:  r.startedAt.UTC(),
		FinishedAt: utcPtr(r.finishedAt),
	}
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
