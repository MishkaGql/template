package trip

import (
	"github.com/MishkaGql/template/api"
	applicationTrip "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/domain/trip"
)

func toInput(request api.TripData) applicationTrip.Input {
	return applicationTrip.Input{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		Price:    request.Price,

		StartPoint: applicationTrip.Coordinates(request.StartPoint),
		EndPoint:   applicationTrip.Coordinates(request.EndPoint),
	}
}

func toResponse(output *trip.Trip) api.Trip {
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

func toPositionInput(request api.PositionData) applicationTrip.PositionInput {
	return applicationTrip.PositionInput{
		Coords:     applicationTrip.Coordinates(request.Coordinates),
		RecordedAt: request.RecordedAt,
	}
}

func toPositionResponse(position *trip.Position) api.TripPosition {
	return api.TripPosition{
		Id:          position.ID,
		TripId:      position.TripID,
		Coordinates: api.Coordinates(position.Coordinates),
		RecordedAt:  position.RecordedAt,
	}
}

func toPositionsResponse(positions []trip.Position) api.TripPositionsResponse {
	response := api.TripPositionsResponse{
		Positions: make([]api.TripPosition, 0, len(positions)),
	}
	for i := range positions {
		response.Positions = append(response.Positions, toPositionResponse(&positions[i]))
	}

	return response
}
