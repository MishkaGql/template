package mapper

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/repository/entity"
)

func ToApi(te entity.TripEntity) api.Trip {
	return api.Trip{
		DriverId: te.DriverID,
		EndPoint: api.Coordinates{
			Latitude:  te.EndLatitude,
			Longitude: te.EndLongitude,
		},
		FinishedAt:     toTimeFromPgTime(te.FinishedAt),
		Id:             te.ID,
		LastPositionAt: nil,
		Price:          te.Price,
		StartPoint: api.Coordinates{
			Latitude:  te.StartLatitude,
			Longitude: te.StartLongitude,
		},
		StartedAt: te.StartedAt,
		Status:    te.Status,
		UserId:    te.UserID,
	}
}

func ToEntity(t api.Trip) entity.TripEntity {
	return entity.TripEntity{
		ID:             t.Id,
		UserID:         t.UserId,
		DriverID:       t.DriverId,
		StartLatitude:  t.StartPoint.Latitude,
		StartLongitude: t.StartPoint.Longitude,
		EndLatitude:    t.EndPoint.Latitude,
		EndLongitude:   t.EndPoint.Longitude,
		Price:          t.Price,
		Status:         t.Status,
		StartedAt:      t.StartedAt,
		FinishedAt:     toPgTypeTime(t.FinishedAt),
	}
}
func toTimeFromPgTime(time pgtype.Timestamptz) *time.Time {
	if !time.Valid {
		return nil
	}
	return &time.Time
}
func toPgTypeTime(time *time.Time) pgtype.Timestamptz {
	if time == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *time, Valid: true}
}
