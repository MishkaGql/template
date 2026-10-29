package trip

import (
	"time"
	"github.com/google/uuid"
)

type Coordinates struct {
	Latitude float64
	Longitude float64 
}

type Status string

const (
	Active Status = "active"
	Completed Status = "completed"
)

type Trip struct {
	ID   			uuid.UUID 
	DriverID 		uuid.UUID
	UserID    		uuid.UUID
	Price 			int64 
	Status     		Status   

	StartPoint 		Coordinates    
	EndPoint		Coordinates       
	LastPositionAt *time.Time   

	FinishedAt     *time.Time           
	StartedAt  		time.Time        
}

func (t *Trip) Complete() {
	t.Status = Completed
	
	now := time.Now()
	t.FinishedAt = &now
}

func New(
	userID uuid.UUID,
	driverID uuid.UUID,
	price int64,
	startPoint Coordinates, 
	endPoint Coordinates,
) *Trip {
	id, _ := uuid.NewV7()

	return &Trip{
		ID: id,
		DriverID: driverID,
		UserID: userID,
		Price: price, 
		Status: Active,
		StartPoint: startPoint,
		EndPoint:	endPoint, 
		LastPositionAt: nil,      
		FinishedAt: nil,           
		StartedAt: time.Now(), 
	}
	
}