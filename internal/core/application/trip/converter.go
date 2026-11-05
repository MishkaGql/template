package trip

import "github.com/MishkaGql/template/internal/core/domain/trip"

func MapToOutput(t *trip.Trip) Output {
	return Output{
		ID:       t.ID,
		UserID:   t.UserID,
		DriverID: t.DriverID,
		Price:    t.Price,
		Status:   Status(t.Status),

		StartPoint: Coordinates(t.StartPoint),
		EndPoint:   Coordinates(t.EndPoint),

		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}
