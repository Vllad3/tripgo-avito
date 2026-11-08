package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	Start      Point
	End        Point
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

func New(id, userID, driverID uuid.UUID, start, end Point, price int64, startedAt time.Time) (*Trip, error) {
	const message = "must not be empty"

	switch {
	case id == uuid.Nil:
		return nil, &ValidationError{Field: "id", Message: message}
	case userID == uuid.Nil:
		return nil, &ValidationError{Field: "user_id", Message: message}
	case driverID == uuid.Nil:
		return nil, &ValidationError{Field: "driver_id", Message: message}
	case price < 0:
		return nil, &ValidationError{Field: "price", Message: "must be >= 0"}
	}

	if err := start.validate("start_point"); err != nil {
		return nil, err
	}
	if err := end.validate("end_point"); err != nil {
		return nil, err
	}

	return &Trip{
		ID:        id,
		UserID:    userID,
		DriverID:  driverID,
		Start:     start,
		End:       end,
		Price:     price,
		Status:    StatusActive,
		StartedAt: startedAt,
	}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}

func (p Point) validate(field string) error {
	if !(p.Latitude >= -90 && p.Latitude <= 90) {
		return &ValidationError{Field: field + ".latitude", Message: "must be in [-90, 90]"}
	}
	if !(p.Longitude >= -180 && p.Longitude <= 180) {
		return &ValidationError{Field: field + ".longitude", Message: "must be in [-180, 180]"}
	}
	return nil
}
