package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	"github.com/Vllad3/tripgo-avito/internal/usecase/trip/get"
)

var (
	tripID = uuid.MustParse("1f0a9c62-4a1c-4f2e-9d33-2a4bb0f0b111")
	errDB  = errors.New("db is down")
)

func TestUseCase_Execute(t *testing.T) {
	found := &trip.Trip{
		ID:        tripID,
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		repoRes *trip.Trip
		repoErr error
		wantErr error
	}{
		{name: "found", repoRes: found},
		{name: "not found", repoErr: trip.ErrTripNotFound, wantErr: trip.ErrTripNotFound},
		{name: "repository error", repoErr: errDB, wantErr: errDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMocktripRepository(gomock.NewController(t))
			repo.EXPECT().Get(gomock.Any(), tripID).Return(tt.repoRes, tt.repoErr)

			got, err := get.New(repo).Execute(context.Background(), tripID)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got != found {
				t.Errorf("trip = %+v, want %+v", got, found)
			}
			if tt.wantErr != nil && got != nil {
				t.Errorf("trip = %+v, want nil on error", got)
			}
		})
	}
}
