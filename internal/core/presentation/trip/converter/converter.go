package converter

import (
	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/application/trip/dto"
)

func ToInput(request api.TripData) dto.TripInput {
	return dto.TripInput{
		UserId:   request.UserId,
		DriverId: request.DriverId,
		Price:	  request.Price,

    	EndPoint: dto.Coordinates(request.EndPoint),
    	StartPoint: dto.Coordinates(request.StartPoint),
	}
}

func ToResponse(output dto.TripOutput) api.Trip {
	return api.Trip{
		Id: 			output.ID,
		UserId: 		output.UserId,
   		DriverId: 		output.DriverID,

		Price: 			output.Price,

		Status: 		api.TripStatus(output.Status),     

   		EndPoint: 		api.Coordinates(output.EndPoint),
		StartPoint: 	api.Coordinates(output.StartPoint),
		LastPositionAt: output.LastPositionAt,  

		StartedAt:  	output.StartedAt,
    	FinishedAt: 	output.FinishedAt,
	}
}

