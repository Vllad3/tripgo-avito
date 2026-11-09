package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	"github.com/Vllad3/tripgo-avito/internal/usecase/trip/finish"
)

var (
	tripID     = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	startedAt  = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	finishedAt = startedAt.Add(18 * time.Minute)
	errDB      = errors.New("db is down")
)

func activeTrip() *trip.Trip {
	return &trip.Trip{ID: tripID, Status: trip.StatusActive, StartedAt: startedAt}
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
	uc      *finish.UseCase
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctrl := gomock.NewController(t)

	f := &fixture{
		trips:   NewMocktripRepository(ctrl),
		history: NewMockhistoryRepository(ctrl),
		tx:      NewMocktxManager(ctrl),
	}
	f.uc = finish.New(f.trips, f.history, f.tx, func() time.Time { return finishedAt })
	return f
}

func TestUseCase_Execute_Success(t *testing.T) {
	f := newFixture(t)
	from := trip.StatusActive

	runInTx(f.tx)
	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *trip.Trip) error {
			if got.Status != trip.StatusCompleted {
				t.Errorf("saved status = %q, want completed", got.Status)
			}
			return nil
		},
	)
	f.history.EXPECT().Add(gomock.Any(), tripID, &from, trip.StatusCompleted).Return(nil)

	got, err := f.uc.Execute(context.Background(), tripID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != trip.StatusCompleted {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finishedAt) {
		t.Errorf("finished_at = %v, want %v", got.FinishedAt, finishedAt)
	}
}

func TestUseCase_Execute_NotFound(t *testing.T) {
	f := newFixture(t)

	runInTx(f.tx)
	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(nil, trip.ErrTripNotFound)

	got, err := f.uc.Execute(context.Background(), tripID)
	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("error = %v, want ErrTripNotFound", err)
	}
	if got != nil {
		t.Errorf("trip = %+v, want nil on error", got)
	}
}

func TestUseCase_Execute_AlreadyCompleted(t *testing.T) {
	f := newFixture(t)

	done := activeTrip()
	if err := done.Finish(startedAt.Add(time.Minute)); err != nil {
		t.Fatalf("prepare completed trip: %v", err)
	}

	runInTx(f.tx)
	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(done, nil)

	got, err := f.uc.Execute(context.Background(), tripID)
	if !errors.Is(err, trip.ErrTripCompleted) {
		t.Fatalf("error = %v, want ErrTripCompleted", err)
	}
	if got != nil {
		t.Errorf("trip = %+v, want nil on error", got)
	}
	if !done.FinishedAt.Equal(startedAt.Add(time.Minute)) {
		t.Errorf("finished_at = %v, must not be overwritten", done.FinishedAt)
	}
}

func TestUseCase_Execute_UpdateFails(t *testing.T) {
	f := newFixture(t)

	runInTx(f.tx)
	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), gomock.Any()).Return(errDB)
	// history.Add не ожидается

	_, err := f.uc.Execute(context.Background(), tripID)
	if !errors.Is(err, errDB) {
		t.Fatalf("error = %v, want errDB", err)
	}
}

func TestUseCase_Execute_HistoryFails(t *testing.T) {
	f := newFixture(t)
	from := trip.StatusActive

	runInTx(f.tx)
	f.trips.EXPECT().GetForUpdate(gomock.Any(), tripID).Return(activeTrip(), nil)
	f.trips.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
	f.history.EXPECT().Add(gomock.Any(), tripID, &from, trip.StatusCompleted).Return(errDB)

	_, err := f.uc.Execute(context.Background(), tripID)
	if !errors.Is(err, errDB) {
		t.Fatalf("error = %v, want errDB", err)
	}
}
