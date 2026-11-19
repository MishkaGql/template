package trip

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
	Status         string             `db:"status"`
	StartedAt      time.Time          `db:"started_at"`
	FinishedAt     pgtype.Timestamptz `db:"finished_at"`
	LastPositionAt pgtype.Timestamptz `db:"last_position_at"`
	CreatedAt      time.Time          `db:"created_at"`
	UpdatedAt      time.Time          `db:"updated_at"`
}

type PositionEntity struct {
	ID         int64     `db:"id"`
	TripID     uuid.UUID `db:"trip_id"`
	Latitude   float64   `db:"latitude"`
	Longitude  float64   `db:"longitude"`
	RecordedAt time.Time `db:"recorded_at"`
	CreatedAt  time.Time `db:"created_at"`
}

type StatusHistoryEntity struct {
	ID         int64       `db:"id"`
	TripID     uuid.UUID   `db:"trip_id"`
	FromStatus pgtype.Text `db:"from_status"`
	ToStatus   string      `db:"to_status"`
	Reason     pgtype.Text `db:"reason"`
	ChangedAt  time.Time   `db:"changed_at"`
}
