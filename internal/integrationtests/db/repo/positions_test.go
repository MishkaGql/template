package repo

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/container"
	service "github.com/MishkaGql/template/internal/core/application/trip"
	"github.com/MishkaGql/template/internal/core/infrastructure/db"
	postgres "github.com/MishkaGql/template/internal/core/infrastructure/trip"
	handler "github.com/MishkaGql/template/internal/core/presentation/trip"
	"github.com/MishkaGql/template/internal/observability"
)

func TestListTripPositions_Success(t *testing.T) {
	f := newPositionsFixture(t)
	body := f.request(t, http.MethodGet, f.tripID, "", http.StatusOK)
	require.JSONEq(t, `{"positions":[]}`, string(body))

	latest := time.Now().UTC().Truncate(time.Microsecond)
	earlier := latest.Add(-time.Hour)

	last := f.savePosition(t, latest)
	first := f.savePosition(t, earlier)
	second := f.savePosition(t, earlier)
	require.Less(t, first.Id, second.Id)

	body = f.request(t, http.MethodGet, f.tripID, "", http.StatusOK)
	var route api.TripPositionsResponse
	require.NoError(t, json.Unmarshal(body, &route))
	require.Equal(t, []api.TripPosition{first, second, last}, route.Positions)

	stored, err := f.repo.Get(t.Context(), f.tripID)
	require.NoError(t, err)
	require.NotNil(t, stored.LastPositionAt)
	require.True(t, stored.LastPositionAt.Equal(latest))
	require.Equal(t, []uuid.UUID{f.tripID, f.tripID, f.tripID}, f.cancelled)
}

func TestCreateTripPosition_InvalidRequest(t *testing.T) {
	f := newPositionsFixture(t)
	now := time.Now().UTC()

	tests := []struct {
		name string
		body string
	}{
		{"latitude out of range", positionBody(91, now.Format(time.RFC3339Nano))},
		{"invalid timestamp", positionBody(10, "invalid")},
		{"timestamp in the future", positionBody(10, now.Add(2*time.Minute).Format(time.RFC3339Nano))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f.request(t, http.MethodPost, f.tripID, tc.body, http.StatusBadRequest)
		})
	}
	require.Empty(t, f.cancelled)
}

func TestTripPositions_TripNotFound(t *testing.T) {
	f := newPositionsFixture(t)
	missingID := uuid.New()
	body := positionBody(10, time.Now().UTC().Format(time.RFC3339Nano))

	f.request(t, http.MethodGet, missingID, "", http.StatusNotFound)
	f.request(t, http.MethodPost, missingID, body, http.StatusNotFound)
	require.Empty(t, f.cancelled)
}

func TestCreateTripPosition_TripCompleted(t *testing.T) {
	f := newPositionsFixture(t)
	trip, err := f.repo.Get(t.Context(), f.tripID)
	require.NoError(t, err)
	require.NoError(t, trip.Complete())
	_, err = f.repo.Update(t.Context(), trip)
	require.NoError(t, err)

	body := positionBody(10, time.Now().UTC().Format(time.RFC3339Nano))
	f.request(t, http.MethodPost, f.tripID, body, http.StatusConflict)
	require.Empty(t, f.cancelled)
}

func TestCreateTripPosition_RollbackOnInsertFailure(t *testing.T) {
	f := newPositionsFixture(t)
	lastSavedAt := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Minute)
	saved := f.savePosition(t, lastSavedAt)

	_, err := pool.Exec(t.Context(), "ALTER TABLE trip_positions ADD CONSTRAINT test_position_failure CHECK (latitude <> 11)")
	require.NoError(t, err)
	defer func() {
		_, err := pool.Exec(t.Context(), "ALTER TABLE trip_positions DROP CONSTRAINT test_position_failure")
		require.NoError(t, err)
	}()

	newer := lastSavedAt.Add(time.Second)
	f.request(t, http.MethodPost, f.tripID, positionBody(11, newer.Format(time.RFC3339Nano)), http.StatusInternalServerError)

	stored, err := f.repo.Get(t.Context(), f.tripID)
	require.NoError(t, err)
	require.NotNil(t, stored.LastPositionAt)
	require.True(t, stored.LastPositionAt.Equal(lastSavedAt))

	positions, err := f.repo.ListTripPositions(t.Context(), f.tripID)
	require.NoError(t, err)
	require.Len(t, positions, 1)
	require.Equal(t, saved.Id, positions[0].ID)
	require.Equal(t, []uuid.UUID{f.tripID}, f.cancelled)
}

type positionsFixture struct {
	router    http.Handler
	repo      *postgres.Repository
	tripID    uuid.UUID
	cancelled []uuid.UUID
}

func newPositionsFixture(t *testing.T) *positionsFixture {
	t.Helper()

	repo := mustNewRepository(t)
	trip, err := repo.Create(t.Context(), mustNewTrip(t, uuid.New()))
	require.NoError(t, err)

	logger := slog.New(slog.DiscardHandler)
	metrics, err := observability.NewMetrics(noop.NewMeterProvider())
	require.NoError(t, err)

	f := &positionsFixture{repo: repo, tripID: trip.ID}
	cancelPending := func(id uuid.UUID) {
		f.cancelled = append(f.cancelled, id)
	}
	svc := service.NewService(repo, db.NewPgxTxManager(pool), logger, metrics)
	h := handler.NewHandler(svc, db.NewPinger(pool, time.Second), logger, cancelPending)
	f.router, err = container.NewRouter(h, logger, metrics)
	require.NoError(t, err)

	return f
}

func (f *positionsFixture) request(t *testing.T, method string, id uuid.UUID, body string, wantStatus int) []byte {
	t.Helper()

	path := "/api/v1/trips/" + id.String() + "/positions"
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, req)
	require.Equal(t, wantStatus, response.Code, "%s %s: %s", method, path, response.Body.String())

	return response.Body.Bytes()
}

func (f *positionsFixture) savePosition(t *testing.T, recordedAt time.Time) api.TripPosition {
	t.Helper()

	body := positionBody(10, recordedAt.Format(time.RFC3339Nano))
	response := f.request(t, http.MethodPost, f.tripID, body, http.StatusCreated)

	var saved api.TripPosition
	require.NoError(t, json.Unmarshal(response, &saved))
	require.Equal(t, f.tripID, saved.TripId)
	require.True(t, saved.RecordedAt.Equal(recordedAt))
	return saved
}

func positionBody(latitude int, recordedAt string) string {
	return fmt.Sprintf(`{"coordinates":{"latitude":%d,"longitude":30},"recorded_at":%q}`, latitude, recordedAt)
}
