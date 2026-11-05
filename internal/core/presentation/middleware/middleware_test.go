package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MishkaGql/template/api"
)

func TestRecover_CodeInternalError_HandlerPanicked(t *testing.T) {
	rec := serveRecovered(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var body api.Problem
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "internal_error", body.Code)
}

func TestRecover_No_HandlerWithoutPanic(t *testing.T) {
	rec := serveRecovered(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestLogging_InfoLevel_Response2xx(t *testing.T) {
	assertLoggedLevel(t, http.StatusCreated, slog.LevelInfo)
}

func TestLogging_InfoLevel_Response4xx(t *testing.T) {
	assertLoggedLevel(t, http.StatusConflict, slog.LevelInfo)
}

func TestLogging_ErrorLevel_Response5xx(t *testing.T) {
	assertLoggedLevel(t, http.StatusInternalServerError, slog.LevelError)
}

func serveRecovered(t *testing.T, handler http.Handler) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)

	Recover(slog.New(slog.DiscardHandler))(handler).ServeHTTP(rec, req)

	return rec
}

func assertLoggedLevel(t *testing.T, status int, want slog.Level) {
	t.Helper()

	recorder := &levelRecorder{}
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", nil)

	Logging("trip-service", slog.New(recorder), func(*http.Request) string { return "/api/v1/trips" })(handler).
		ServeHTTP(rec, req)

	require.Len(t, recorder.records, 1)
	record := recorder.records[0]

	assert.Equal(t, want, record.Level)
	assert.Equal(t, "request handled", record.Message)
	assert.Equal(t, "/api/v1/trips", attrOf(t, record, "route"))
	assert.Equal(t, http.MethodPost, attrOf(t, record, "method"))
	assert.Equal(t, "trip-service", attrOf(t, record, "service"))
}

func attrOf(t *testing.T, record slog.Record, key string) string {
	t.Helper()

	var value string

	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key == key {
			value = attr.Value.String()

			return false
		}

		return true
	})

	return value
}

type levelRecorder struct {
	records []slog.Record
}

func (r *levelRecorder) Enabled(context.Context, slog.Level) bool {
	return true
}

func (r *levelRecorder) Handle(_ context.Context, record slog.Record) error {
	r.records = append(r.records, record)

	return nil
}

func (r *levelRecorder) WithAttrs(_ []slog.Attr) slog.Handler {
	return r
}

func (r *levelRecorder) WithGroup(_ string) slog.Handler {
	return r
}
