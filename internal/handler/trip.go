package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Vllad3/tripgo-avito/internal/domain"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
)

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var body api.CreateTripJSONRequestBody

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Invalid request body")
		return
	}

	if err := validateTripData(body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	trip := &domain.Trip{
		ID:             uuid.New(),
		UserID:         body.UserId,
		DriverID:       body.DriverId,
		StartLatitude:  body.StartPoint.Latitude,
		StartLongitude: body.StartPoint.Longitude,
		EndLatitude:    body.EndPoint.Latitude,
		EndLongitude:   body.EndPoint.Longitude,
		Price:          body.Price,
		Status:         domain.TripStatusActive,
		StartedAt:      time.Now().UTC(),
	}

	err := h.tx.Do(r.Context(), func(ctx context.Context) error {
		if err := h.trips.CreateTrip(ctx, trip); err != nil {
			return err
		}
		return h.history.Create(ctx, trip.ID, nil, domain.TripStatusActive, nil)
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	if err := writeJSON(w, http.StatusCreated, toAPITrip(trip)); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.trips.GetById(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, toAPITrip(trip)); err != nil {
		log.Printf("write response: %v", err)
	}
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	var trip *domain.Trip

	err := h.tx.Do(r.Context(), func(ctx context.Context) error {
		if err := h.trips.Finish(ctx, tripId); err != nil {
			return err
		}

		from := domain.TripStatusActive
		if err := h.history.Create(ctx, tripId, &from, domain.TripStatusCompleted, nil); err != nil {
			return err
		}

		t, err := h.trips.GetById(ctx, tripId)
		if err != nil {
			return err
		}
		trip = t
		return nil
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := writeJSON(w, http.StatusOK, toAPITrip(trip)); err != nil {
		log.Printf("write response: %v", err)
	}
}

func validateTripData(d api.TripData) error {
	if d.UserId == uuid.Nil {
		return errors.New("user_id is required")
	}
	if d.DriverId == uuid.Nil {
		return errors.New("driver_id is required")
	}
	if d.Price < 0 {
		return errors.New("price must be >= 0")
	}
	if err := validateCoordinates("start_point", d.StartPoint); err != nil {
		return err
	}
	if err := validateCoordinates("end_point", d.EndPoint); err != nil {
		return err
	}
	return nil
}

func validateCoordinates(field string, c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return fmt.Errorf("%s.latitude must be in [-90, 90]", field)
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return fmt.Errorf("%s.longitude must be in [-180, 180]", field)
	}
	return nil
}

func toAPITrip(t *domain.Trip) api.Trip {
	var finishedAt *time.Time
	if t.FinishedAt != nil {
		f := t.FinishedAt.UTC()
		finishedAt = &f
	}

	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartLatitude, Longitude: t.StartLongitude},
		EndPoint:   api.Coordinates{Latitude: t.EndLatitude, Longitude: t.EndLongitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt.UTC(),
		FinishedAt: finishedAt,
	}
}
