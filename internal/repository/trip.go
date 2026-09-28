package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Masterminds/squirrel"
	"github.com/Vllad3/tripgo-avito/internal/database"
	"github.com/Vllad3/tripgo-avito/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTripNotFound         = errors.New("trip not found")
	ErrTripAlreadyCompleted = errors.New("trip already completed")
	ErrDriverBusy           = errors.New("driver already has an active trip")
)

const activeTripPerDriverIndex = "trips_unique_driver_id_idx"

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *domain.Trip) error
	GetById(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) error
}

type tripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) TripRepository {
	return &tripRepository{pool: pool}
}

func (r *tripRepository) CreateTrip(ctx context.Context, trip *domain.Trip) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		Values(trip.ID, trip.UserID, trip.DriverID, trip.StartLatitude, trip.StartLongitude, trip.EndLatitude, trip.EndLongitude, trip.Price, trip.Status, trip.StartedAt, trip.FinishedAt).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = r.db(ctx).Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == activeTripPerDriverIndex {
			return ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *tripRepository) GetById(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	query, args, err := squirrel.
		Select("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var trip domain.Trip
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartLatitude,
		&trip.StartLongitude,
		&trip.EndLatitude,
		&trip.EndLongitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTripNotFound
		}
		return nil, fmt.Errorf("query row: %w", err)
	}

	return &trip, nil
}

func (r *tripRepository) Finish(ctx context.Context, id uuid.UUID) error {
	query, args, err := squirrel.
		Update("trips").
		Set("status", domain.TripStatusCompleted).
		Set("finished_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": domain.TripStatusActive}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	cmdTag, err := r.db(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		isExists, err := r.existsTrip(ctx, id)
		if err != nil {
			return fmt.Errorf("check trip exists: %w", err)
		}
		if !isExists {
			return ErrTripNotFound
		}

		return ErrTripAlreadyCompleted
	}

	return nil
}

func (r *tripRepository) existsTrip(ctx context.Context, id uuid.UUID) (bool, error) {
	query, args, err := squirrel.
		Select("1").
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build query: %w", err)
	}

	var one int
	err = r.db(ctx).QueryRow(ctx, query, args...).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query row: %w", err)
	}

	return true, nil
}

func (r *tripRepository) db(ctx context.Context) database.Executor {
	return database.ExecutorFromContext(ctx, r.pool)
}
