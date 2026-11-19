package repo

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/MishkaGql/template/internal/core/domain/trip"
	postgres "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	"github.com/MishkaGql/template/internal/worker/positions"
)

func TestFindStale(t *testing.T) {
	repo := mustNewRepository(t)
	now := time.Now()

	started := now.Add(-6 * time.Hour)
	stale := now.Add(-2 * time.Hour)
	fresh := now.Add(-time.Second)

	withStalePosition := insertTripForScan(t, repo, trip.StatusActive, started, &stale)
	withoutPositions := insertTripForScan(t, repo, trip.StatusActive, started, nil)
	insertTripForScan(t, repo, trip.StatusActive, started, &fresh)
	insertTripForScan(t, repo, trip.StatusActive, fresh, nil)
	insertTripForScan(t, repo, trip.StatusCompleted, started, nil)

	got, err := repo.FindStale(t.Context(), time.Minute)
	require.NoError(t, err)
	require.Equal(t, []positions.Trip{withoutPositions, withStalePosition}, got)
}

func insertTripForScan(t *testing.T, repo *postgres.Repository, status trip.Status, startedAt time.Time, lastPositionAt *time.Time) positions.Trip {
	t.Helper()

	tr := mustNewTrip(t, uuid.New())
	tr.StartedAt = startedAt
	if status == trip.StatusCompleted {
		require.NoError(t, tr.Complete())
	}

	_, err := repo.Create(t.Context(), tr)
	require.NoError(t, err)

	if lastPositionAt != nil {
		_, err := pool.Exec(t.Context(), "UPDATE trips SET last_position_at=$1 WHERE id=$2", lastPositionAt, tr.ID)
		require.NoError(t, err)
	}

	return positions.Trip{ID: tr.ID, DriverID: tr.DriverID}
}
