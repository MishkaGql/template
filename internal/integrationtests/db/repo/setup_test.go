package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	infrastructureTrip "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	"github.com/MishkaGql/template/internal/integrationtests/db/dbtest"
)

func mustNewRepository(t *testing.T) *infrastructureTrip.Repository {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), dbtest.QueryTimeout)
	defer cancel()

	if _, err := pool.Exec(ctx, `TRUNCATE trips, trip_status_history RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate test tables: unexpected error: %v", err)
	}

	return infrastructureTrip.NewRepository(pool, dbtest.QueryTimeout)
}

func mustNewTrip(t *testing.T, driverID uuid.UUID) *trip.Trip {
	t.Helper()

	tr, err := trip.New(
		uuid.New(),
		driverID,
		1450,
		trip.Coordinates{Latitude: 59.9398, Longitude: 30.3146},
		trip.Coordinates{Latitude: 59.929, Longitude: 30.3626},
	)
	if err != nil {
		t.Fatalf("New(driverID=%s): unexpected error: %v", driverID, err)
	}

	return tr
}

func mustCountHistory(t *testing.T, id uuid.UUID) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), dbtest.QueryTimeout)
	defer cancel()

	var count int

	err := pool.
		QueryRow(ctx, `SELECT count(*) FROM trip_status_history WHERE trip_id = $1`, id).
		Scan(&count)
	if err != nil {
		t.Fatalf("count history(%s): unexpected error: %v", id, err)
	}

	return count
}
