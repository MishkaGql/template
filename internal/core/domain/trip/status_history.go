package trip

import "github.com/google/uuid"

type StatusHistory struct {
	TripID     uuid.UUID
	FromStatus *Status
	ToStatus   Status
}

func NewStatusHistory(tripID uuid.UUID, from *Status, to Status) *StatusHistory {
	return &StatusHistory{
		TripID:     tripID,
		FromStatus: from,
		ToStatus:   to,
	}
}
