package converter

import (
	"github.com/MishkaGql/template/internal/core/application/trip/dto"
	"github.com/MishkaGql/template/internal/core/domain/trip"
)

func ToOutput(trip trip.Trip) dto.TripOutput {
	return dto.TripOutput{
		ID: 			trip.ID,
		UserId: 		trip.UserID,
   		DriverID: 		trip.DriverID,

		Price: 			trip.Price,

		Status: 		dto.Status(trip.Status),     

   		EndPoint: 		dto.Coordinates(trip.EndPoint),
		StartPoint: 	dto.Coordinates(trip.StartPoint),
		LastPositionAt: trip.LastPositionAt,  

		StartedAt:  	trip.StartedAt,
    	FinishedAt: 	trip.FinishedAt,
	}
} 