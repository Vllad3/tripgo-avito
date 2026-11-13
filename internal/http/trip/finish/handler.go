package finish

import (
	"net/http"

	"github.com/google/uuid"

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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID) {
	finished, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		h.errs.Error(w, r, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.ToAPI(finished))
}
