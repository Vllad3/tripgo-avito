package create

import (
	"encoding/json"
	"net/http"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
	"github.com/Vllad3/tripgo-avito/internal/http/domainerror"
	"github.com/Vllad3/tripgo-avito/internal/http/response"
	"github.com/Vllad3/tripgo-avito/internal/http/trip/dto"
)

type Handler struct {
	uc   useCase
	errs *domainerror.Writer
}

func New(uc useCase, errs *domainerror.Writer) *Handler {
	return &Handler{uc: uc, errs: errs}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var body api.CreateTripJSONRequestBody

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		h.errs.Error(w, r, &trip.ValidationError{
			Field:   "body",
			Message: "must be a valid JSON object without unknown fields",
		})
		return
	}

	created, err := h.uc.Execute(r.Context(), toInput(body))
	if err != nil {
		h.errs.Error(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	response.JSON(w, http.StatusCreated, dto.ToAPI(created))
}
