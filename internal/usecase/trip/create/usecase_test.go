package create_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	"github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
)

var (
	tripID = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	now    = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	errDB  = errors.New("db is down")
)

func validInput() create.Input {
	return create.Input{
		UserID:   uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
		DriverID: uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
		Start:    trip.Point{Latitude: 59.9398, Longitude: 30.3146},
		End:      trip.Point{Latitude: 59.9290, Longitude: 30.3626},
		Price:    1450,
	}
}

func wantTrip(in create.Input) *trip.Trip {
	return &trip.Trip{
		ID:        tripID,
		UserID:    in.UserID,
		DriverID:  in.DriverID,
		Start:     in.Start,
		End:       in.End,
		Price:     in.Price,
		Status:    trip.StatusActive,
		StartedAt: now,
	}
}

func runInTx(tx *MocktxManager) {
	tx.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) },
	)
}

type fixture struct {
	trips   *MocktripRepository
	history *MockhistoryRepository
	tx      *MocktxManager
	metrics *Mockmetrics
	uc      *create.UseCase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	f := &fixture{
		trips:   NewMocktripRepository(ctrl),
		history: NewMockhistoryRepository(ctrl),
		tx:      NewMocktxManager(ctrl),
		metrics: NewMockmetrics(ctrl),
	}
	f.uc = create.New(f.trips, f.history, f.tx,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		f.metrics,
		func() time.Time { return now },
		func() uuid.UUID { return tripID },
	)
	return f
}

func TestUseCase_Execute_Success(t *testing.T) {
	f := newFixture(t)
	in := validInput()

	runInTx(f.tx)
	f.trips.EXPECT().Save(gomock.Any(), wantTrip(in)).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), tripID, gomock.Nil(), trip.StatusActive).Return(nil)
	f.metrics.EXPECT().TripCreated(gomock.Any())

	got, err := f.uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != tripID || got.Status != trip.StatusActive {
		t.Errorf("trip = %+v, want id %s with status active", got, tripID)
	}
}

func TestUseCase_Execute_DriverBusy(t *testing.T) {
	f := newFixture(t)
	in := validInput()

	runInTx(f.tx)
	f.trips.EXPECT().Save(gomock.Any(), wantTrip(in)).Return(trip.ErrDriverBusy)

	got, err := f.uc.Execute(context.Background(), in)
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("error = %v, want ErrDriverBusy", err)
	}
	if got != nil {
		t.Errorf("trip = %+v, want nil on error", got)
	}
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.Price = -1

	_, err := f.uc.Execute(context.Background(), in)

	var vErr *trip.ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("error = %v, want *ValidationError", err)
	}
	if vErr.Field != "price" {
		t.Errorf("field = %q, want price", vErr.Field)
	}
}

func TestUseCase_Execute_HistoryFails(t *testing.T) {
	f := newFixture(t)
	in := validInput()

	runInTx(f.tx)
	f.trips.EXPECT().Save(gomock.Any(), wantTrip(in)).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), tripID, gomock.Nil(), trip.StatusActive).Return(errDB)

	got, err := f.uc.Execute(context.Background(), in)
	if !errors.Is(err, errDB) {
		t.Fatalf("error = %v, want errDB", err)
	}
	if got != nil {
		t.Errorf("trip = %+v, want nil on error", got)
	}
}

func TestUseCase_Execute_TxFails(t *testing.T) {
	f := newFixture(t)

	f.tx.EXPECT().Do(gomock.Any(), gomock.Any()).Return(errDB)

	_, err := f.uc.Execute(context.Background(), validInput())
	if !errors.Is(err, errDB) {
		t.Fatalf("error = %v, want errDB", err)
	}
}
