package dto

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	Active Status = "active"
	completed Status = "completed"
)

type (

	Coordinates struct {
		Latitude float64 

		Longitude float64 
	}

	TripInput struct {
		UserId     	uuid.UUID
		DriverId 	uuid.UUID  

		Price 		int64 

		EndPoint 	Coordinates 
		StartPoint 	Coordinates   
	}

	TripOutput struct {
		ID   			uuid.UUID
		UserId   		uuid.UUID 
		DriverID 		uuid.UUID

		Price 			int64 

		Status     		Status   

		StartPoint 		Coordinates    
		EndPoint		Coordinates       
		LastPositionAt 	*time.Time   

		FinishedAt     	*time.Time           
		StartedAt  		time.Time  
	}

)

