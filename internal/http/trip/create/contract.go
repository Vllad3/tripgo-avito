package create

import (
	"context"

	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	createuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
)

//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

type useCase interface {
	Execute(ctx context.Context, in createuc.Input) (*trip.Trip, error)
}
