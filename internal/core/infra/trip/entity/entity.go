package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type TripEntity struct {
	ID             uuid.UUID          `db:"id"`
	UserID         uuid.UUID          `db:"user_id"`
	DriverID       uuid.UUID          `db:"driver_id"`
	StartLatitude  float64            `db:"start_latitude"`
	StartLongitude float64            `db:"start_longitude"`
	EndLatitude    float64            `db:"end_latitude"`
	EndLongitude   float64            `db:"end_longitude"`
	Price          int64              `db:"price"`
	Status         string    		  `db:"status"`
	StartedAt      time.Time          `db:"started_at"`
	FinishedAt     pgtype.Timestamptz `db:"finished_at"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
}
