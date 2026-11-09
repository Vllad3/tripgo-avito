//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/Vllad3/tripgo-avito/internal/domain/trip"
	"github.com/Vllad3/tripgo-avito/internal/infra/postgres"
	"github.com/Vllad3/tripgo-avito/internal/infra/postgres/transaction"
	repo "github.com/Vllad3/tripgo-avito/internal/repository/trip"
)

type env struct {
	pool *pgxpool.Pool
	tx   *transaction.Manager
	repo *repo.Repository
}

func newEnv(t *testing.T) *env {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is not set: run `tripgoctl environment start` and `make migrate`")
	}

	pool, err := postgres.NewPool(context.Background(), url, 10, 2, 30*time.Minute, 5*time.Second)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	t.Cleanup(pool.Close)

	return &env{
		pool: pool,
		tx:   transaction.New(pool, 3*time.Second),
		repo: repo.New(pool, 3*time.Second),
	}
}

func (e *env) newTrip(t *testing.T, driverID uuid.UUID) *domain.Trip {
	t.Helper()

	tr, err := domain.New(
		uuid.New(), uuid.New(), driverID,
		domain.Point{Latitude: 59.9398, Longitude: 30.3146},
		domain.Point{Latitude: 59.9290, Longitude: 30.3626},
		1450,
		time.Now().UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		t.Fatalf("build trip: %v", err)
	}

	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", tr.ID)
	})

	return tr
}

func TestRepository_SaveAndGet(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	want := e.newTrip(t, uuid.New())

	if err := e.repo.Save(ctx, want); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := e.repo.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != want.ID || got.UserID != want.UserID || got.DriverID != want.DriverID {
		t.Errorf("ids = %v/%v/%v, want %v/%v/%v", got.ID, got.UserID, got.DriverID, want.ID, want.UserID, want.DriverID)
	}
	if got.Start != want.Start || got.End != want.End {
		t.Errorf("points = %+v/%+v, want %+v/%+v", got.Start, got.End, want.Start, want.End)
	}
	if got.Price != want.Price || got.Status != domain.StatusActive {
		t.Errorf("price/status = %d/%q, want %d/active", got.Price, got.Status, want.Price)
	}
	if !got.StartedAt.Equal(want.StartedAt) {
		t.Errorf("started_at = %v, want %v", got.StartedAt, want.StartedAt)
	}
	if got.FinishedAt != nil {
		t.Errorf("finished_at = %v, want nil", got.FinishedAt)
	}
}

func TestRepository_Get_NotFound(t *testing.T) {
	e := newEnv(t)

	_, err := e.repo.Get(context.Background(), uuid.New())
	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("error = %v, want ErrTripNotFound", err)
	}
}

func TestRepository_Save_DriverBusy(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	driverID := uuid.New()
	first := e.newTrip(t, driverID)
	second := e.newTrip(t, driverID)

	if err := e.repo.Save(ctx, first); err != nil {
		t.Fatalf("save first: %v", err)
	}

	if err := e.repo.Save(ctx, second); !errors.Is(err, domain.ErrDriverBusy) {
		t.Fatalf("save second while first is active: error = %v, want ErrDriverBusy", err)
	}

	if err := first.Finish(time.Now().UTC()); err != nil {
		t.Fatalf("finish first: %v", err)
	}
	if err := e.repo.Update(ctx, first); err != nil {
		t.Fatalf("update first: %v", err)
	}
	if err := e.repo.Save(ctx, second); err != nil {
		t.Fatalf("save second after first is completed: %v", err)
	}
}

func TestTransaction_RollsBackBothInserts(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	tr := e.newTrip(t, uuid.New())

	err := e.tx.Do(ctx, func(ctx context.Context) error {
		if err := e.repo.Save(ctx, tr); err != nil {
			return err
		}

		return e.repo.Add(ctx, tr.ID, nil, domain.Status("broken"))
	})
	if err == nil {
		t.Fatal("expected an error from the history insert")
	}

	if _, err := e.repo.Get(ctx, tr.ID); !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("trip after rollback: error = %v, want ErrTripNotFound", err)
	}
}

func TestRepository_ConcurrentFinish(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	tr := e.newTrip(t, uuid.New())

	if err := e.repo.Save(ctx, tr); err != nil {
		t.Fatalf("save: %v", err)
	}

	const workers = 5
	results := make(chan error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- e.tx.Do(ctx, func(ctx context.Context) error {
				cur, err := e.repo.GetForUpdate(ctx, tr.ID)
				if err != nil {
					return err
				}
				if err := cur.Finish(time.Now().UTC()); err != nil {
					return err
				}
				return e.repo.Update(ctx, cur)
			})
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var succeeded, completed int
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, domain.ErrTripCompleted):
			completed++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if succeeded != 1 || completed != workers-1 {
		t.Errorf("succeeded = %d, completed = %d, want 1 and %d", succeeded, completed, workers-1)
	}

	got, err := e.repo.Get(ctx, tr.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.StatusCompleted || got.FinishedAt == nil {
		t.Errorf("trip = %+v, want completed with finished_at", got)
	}
}
