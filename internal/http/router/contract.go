package router

import (
	"net/http"

	"github.com/google/uuid"
)

type (
	createHandler interface {
		Handle(w http.ResponseWriter, r *http.Request)
	}

	getHandler interface {
		Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID)
	}

	finishHandler interface {
		Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID)
	}

	healthHandler interface {
		Health(w http.ResponseWriter, r *http.Request)
		Ready(w http.ResponseWriter, r *http.Request)
	}
)
