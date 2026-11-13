package trip_test

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
)

type args struct {
	id, userID, driverID uuid.UUID
	start, end           trip.Point
	price                int64
}

func validArgs() args {
	return args{
		id:       uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111"),
		userID:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
		driverID: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
		start:    trip.Point{Latitude: 59.9398, Longitude: 30.3146},
		end:      trip.Point{Latitude: 59.9290, Longitude: 30.3626},
		price:    1450,
	}
}

func TestNew(t *testing.T) {
	startedAt := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		mutate    func(*args)
		wantField string
	}{
		{name: "valid"},
		{name: "zero price is allowed", mutate: func(a *args) { a.price = 0 }},
		{name: "boundary coordinates are allowed", mutate: func(a *args) {
			a.start = trip.Point{Latitude: -90, Longitude: 180}
			a.end = trip.Point{Latitude: 90, Longitude: -180}
		}},
		{name: "empty id", mutate: func(a *args) { a.id = uuid.Nil }, wantField: "id"},
		{name: "empty user_id", mutate: func(a *args) { a.userID = uuid.Nil }, wantField: "user_id"},
		{name: "empty driver_id", mutate: func(a *args) { a.driverID = uuid.Nil }, wantField: "driver_id"},
		{name: "negative price", mutate: func(a *args) { a.price = -1 }, wantField: "price"},
		{name: "start latitude above range", mutate: func(a *args) { a.start.Latitude = 90.0001 }, wantField: "start_point.latitude"},
		{name: "start latitude is NaN", mutate: func(a *args) { a.start.Latitude = math.NaN() }, wantField: "start_point.latitude"},
		{name: "start longitude below range", mutate: func(a *args) { a.start.Longitude = -180.1 }, wantField: "start_point.longitude"},
		{name: "end latitude below range", mutate: func(a *args) { a.end.Latitude = -90.5 }, wantField: "end_point.latitude"},
		{name: "end longitude above range", mutate: func(a *args) { a.end.Longitude = 181 }, wantField: "end_point.longitude"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := validArgs()
			if tt.mutate != nil {
				tt.mutate(&a)
			}

			got, err := trip.New(a.id, a.userID, a.driverID, a.start, a.end, a.price, startedAt)

			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Status != trip.StatusActive {
					t.Errorf("status = %q, want %q", got.Status, trip.StatusActive)
				}
				if got.FinishedAt != nil {
					t.Errorf("finished_at = %v, want nil", got.FinishedAt)
				}
				return
			}

			var vErr *trip.ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("error = %v, want *ValidationError", err)
			}
			if vErr.Field != tt.wantField {
				t.Errorf("field = %q, want %q", vErr.Field, tt.wantField)
			}
			if got != nil {
				t.Errorf("trip = %+v, want nil on error", got)
			}
		})
	}
}

func TestTrip_Finish(t *testing.T) {
	startedAt := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	firstFinish := startedAt.Add(18 * time.Minute)
	secondFinish := firstFinish.Add(time.Minute)

	t.Run("active trip becomes completed", func(t *testing.T) {
		tr := &trip.Trip{Status: trip.StatusActive, StartedAt: startedAt}

		if err := tr.Finish(firstFinish); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tr.Status != trip.StatusCompleted {
			t.Errorf("status = %q, want %q", tr.Status, trip.StatusCompleted)
		}
		if tr.FinishedAt == nil || !tr.FinishedAt.Equal(firstFinish) {
			t.Errorf("finished_at = %v, want %v", tr.FinishedAt, firstFinish)
		}
	})

	t.Run("completed trip is not finished again and is not modified", func(t *testing.T) {
		tr := &trip.Trip{Status: trip.StatusActive, StartedAt: startedAt}
		if err := tr.Finish(firstFinish); err != nil {
			t.Fatalf("first finish: %v", err)
		}

		err := tr.Finish(secondFinish)

		if !errors.Is(err, trip.ErrTripCompleted) {
			t.Fatalf("error = %v, want ErrTripCompleted", err)
		}
		if tr.FinishedAt == nil || !tr.FinishedAt.Equal(firstFinish) {
			t.Errorf("finished_at = %v, want %v (must not be overwritten)", tr.FinishedAt, firstFinish)
		}
	})
}
