package trip

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/MishkaGql/template/internal/core/domain/trip"
)

func MapToTripDomain(t TripEntity) *trip.Trip {
	return &trip.Trip{
		ID:       t.ID,
		UserID:   t.UserID,
		DriverID: t.DriverID,
		Price:    t.Price,
		Status:   trip.Status(t.Status),

		StartPoint: trip.Coordinates{
			Latitude:  t.StartLatitude,
			Longitude: t.StartLongitude,
		},
		EndPoint: trip.Coordinates{
			Latitude:  t.EndLatitude,
			Longitude: t.EndLongitude,
		},

		StartedAt:  t.StartedAt,
		FinishedAt: toTime(t.FinishedAt),
	}
}

func MapToTripEntity(t *trip.Trip) TripEntity {
	return TripEntity{
		ID:       t.ID,
		UserID:   t.UserID,
		DriverID: t.DriverID,
		Price:    t.Price,
		Status:   string(t.Status),

		StartLatitude:  t.StartPoint.Latitude,
		StartLongitude: t.StartPoint.Longitude,
		EndLatitude:    t.EndPoint.Latitude,
		EndLongitude:   t.EndPoint.Longitude,

		StartedAt:  t.StartedAt,
		FinishedAt: toPgxTime(t.FinishedAt),
	}
}

func MapToStatusHistoryEntity(h *trip.StatusHistory) StatusHistoryEntity {
	entity := StatusHistoryEntity{
		TripID:   h.TripID,
		ToStatus: string(h.ToStatus),
	}

	if h.FromStatus != nil {
		entity.FromStatus = pgtype.Text{String: string(*h.FromStatus), Valid: true}
	}

	return entity
}

func toTime(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func toPgxTime(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
