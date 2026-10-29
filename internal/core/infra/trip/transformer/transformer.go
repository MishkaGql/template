package transformer

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	"github.com/MishkaGql/template/internal/core/infra/trip/entity"
)

func ToDomain(te entity.TripEntity) trip.Trip {
	return trip.Trip{
		ID:             te.ID,
		DriverID: 		te.DriverID,
		UserID:    		te.UserID,

		Price:          te.Price,

		Status:    trip.Status(te.Status),
		
		StartPoint: trip.Coordinates{
			Latitude:  te.StartLatitude,
			Longitude: te.StartLongitude,
		},
		EndPoint: trip.Coordinates{
			Latitude:  te.EndLatitude,
			Longitude: te.EndLongitude,
		},
		LastPositionAt: nil,

		StartedAt: 		te.StartedAt,
		FinishedAt:     toTime(te.FinishedAt),
	}
}

func ToEntity(t trip.Trip) entity.TripEntity {
	return entity.TripEntity{
		ID:             t.ID,
		UserID:         t.UserID,
		DriverID:       t.DriverID,
		StartLatitude:  t.StartPoint.Latitude,
		StartLongitude: t.StartPoint.Longitude,
		EndLatitude:    t.EndPoint.Latitude,
		EndLongitude:   t.EndPoint.Longitude,
		Price:          t.Price,
		Status:         string(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     toPgxTime(t.FinishedAt),
	}
}

func toTime(time pgtype.Timestamptz) *time.Time {
	if !time.Valid {
		return nil
	}
	return &time.Time
}

func toPgxTime(time *time.Time) pgtype.Timestamptz {
	if time == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *time, Valid: true}
}
