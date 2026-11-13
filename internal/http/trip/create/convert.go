package create

import (
	"github.com/Vllad3/tripgo-avito/internal/domain/trip"
	api "github.com/Vllad3/tripgo-avito/internal/generated"
	createuc "github.com/Vllad3/tripgo-avito/internal/usecase/trip/create"
)

func toInput(body api.CreateTripJSONRequestBody) createuc.Input {
	return createuc.Input{
		UserID:   body.UserId,
		DriverID: body.DriverId,
		Start:    trip.Point{Latitude: body.StartPoint.Latitude, Longitude: body.StartPoint.Longitude},
		End:      trip.Point{Latitude: body.EndPoint.Latitude, Longitude: body.EndPoint.Longitude},
		Price:    body.Price,
	}
}
