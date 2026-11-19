package repo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/container"
	service "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	handler "github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/observability"
	"github.com/MishkaGql/template/internal/worker/positions"
)

func TestFindStale(t *testing.T) {
	repo := mustNewRepository(t)
	now := time.Now()
	var want []positions.Trip
	for _, tc := range []struct {
		started         time.Duration
		last            time.Duration
		finished, stale bool
	}{
		{-5 * time.Hour, 0, false, true},
		{-6 * time.Hour, -2 * time.Hour, false, true},
		{-6 * time.Hour, -time.Second, false, false},
		{-time.Second, 0, false, false},
		{-7 * time.Hour, 0, true, false},
	} {
		tr := mustNewTrip(t, uuid.New())
		tr.StartedAt = now.Add(tc.started)
		if tc.finished {
			require.NoError(t, tr.Complete())
		}
		_, err := repo.Create(t.Context(), tr)
		require.NoError(t, err)
		if tc.last != 0 {
			_, err = pool.Exec(t.Context(), "UPDATE trips SET last_position_at=$1 WHERE id=$2", now.Add(tc.last), tr.ID)
			require.NoError(t, err)
		}
		if tc.stale {
			want = append(want, positions.Trip{ID: tr.ID, DriverID: tr.DriverID})
		}
	}
	got, err := repo.FindStale(t.Context(), time.Minute)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestPositions_HTTPAndRollback(t *testing.T) {
	repo := mustNewRepository(t)
	tr := mustNewTrip(t, uuid.New())
	_, err := repo.Create(t.Context(), tr)
	require.NoError(t, err)
	metrics, err := observability.NewMetrics(noop.NewMeterProvider())
	require.NoError(t, err)
	logger := slog.New(slog.DiscardHandler)
	svc := service.NewService(repo, db.NewPgxTxManager(pool), logger, metrics)
	var cancelled []uuid.UUID
	h := handler.NewHandler(svc, db.NewPinger(pool, time.Second), logger, func(id uuid.UUID) { cancelled = append(cancelled, id) })
	router, err := container.NewRouter(h, logger, metrics)
	require.NoError(t, err)
	url := "/api/v1/trips/" + tr.ID.String() + "/positions"
	call := func(method, path, body string, status int) []byte {
		t.Helper()
		r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		require.Equal(t, status, w.Code, w.Body.String())
		return w.Body.Bytes()
	}
	point := func(lat int, at string) string {
		return fmt.Sprintf(`{"coordinates":{"latitude":%d,"longitude":30},"recorded_at":%q}`, lat, at)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	stamp := now.Format(time.RFC3339Nano)
	older := now.Add(-time.Hour).Format(time.RFC3339Nano)
	call("GET", url, "", 200)
	missing := "/api/v1/trips/" + uuid.NewString() + "/positions"
	call("GET", missing, "", 404)
	call("POST", missing, point(10, stamp), 404)
	call("POST", url, point(91, stamp), 400)
	call("POST", url, point(10, "invalid"), 400)
	call("POST", url, point(10, now.Add(2*time.Minute).Format(time.RFC3339Nano)), 400)
	for _, at := range []string{stamp, older, older} {
		call("POST", url, point(10, at), http.StatusCreated)
	}
	require.Equal(t, []uuid.UUID{tr.ID, tr.ID, tr.ID}, cancelled)

	_, err = pool.Exec(t.Context(), "ALTER TABLE trip_positions ADD CONSTRAINT test_position_failure CHECK (latitude <> 11)")
	require.NoError(t, err)
	defer func() {
		_, e := pool.Exec(t.Context(), "ALTER TABLE trip_positions DROP CONSTRAINT test_position_failure")
		require.NoError(t, e)
	}()
	call("POST", url, point(11, now.Add(time.Second).Format(time.RFC3339Nano)), 500)
	stored, err := repo.Get(t.Context(), tr.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.LastPositionAt)
	require.True(t, stored.LastPositionAt.Equal(now))
	var route api.TripPositionsResponse
	require.NoError(t, json.Unmarshal(call("GET", url, "", 200), &route))
	require.Len(t, route.Positions, 3)
	require.Less(t, route.Positions[0].Id, route.Positions[1].Id)
	require.True(t, route.Positions[0].RecordedAt.Equal(now.Add(-time.Hour)))
	require.True(t, route.Positions[2].RecordedAt.Equal(now))
	require.Len(t, cancelled, 3)
	_, err = svc.Finish(t.Context(), tr.ID)
	require.NoError(t, err)
	call("POST", url, point(10, stamp), 409)
}
