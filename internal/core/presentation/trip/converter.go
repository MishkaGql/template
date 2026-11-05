package trip

import (
	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/application/trip"
)

func MapToInput(request api.TripData) trip.Input {
	return trip.Input{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		Price:    request.Price,

		StartPoint: trip.Coordinates(request.StartPoint),
		EndPoint:   trip.Coordinates(request.EndPoint),
	}
}

func MapToResponse(output trip.Output) api.Trip {
	return api.Trip{
		Id:       output.ID,
		UserId:   output.UserID,
		DriverId: output.DriverID,
		Price:    output.Price,
		Status:   api.TripStatus(output.Status),

		StartPoint: api.Coordinates(output.StartPoint),
		EndPoint:   api.Coordinates(output.EndPoint),

		StartedAt:      output.StartedAt,
		FinishedAt:     output.FinishedAt,
		LastPositionAt: output.LastPositionAt,
	}
}
