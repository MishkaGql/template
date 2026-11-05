package trip

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/MishkaGql/template/api"
)

func TestHealth_No_CorrectData(t *testing.T) {
	ctrl := gomock.NewController(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	NewHandler(NewMockService(ctrl), NewMockPinger(ctrl), slog.New(slog.DiscardHandler)).Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type: got %q, want %q", ct, "application/json")
	}

	var body api.HealthResponse
	decodeJSON(t, rec, &body)

	if body.Status != api.Ok {
		t.Errorf("статус в теле: got %q, want %q", body.Status, api.Ok)
	}
}

func TestReady_No_DatabaseAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinger := NewMockPinger(ctrl)
	pinger.EXPECT().Ping(gomock.Any()).Return(nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	NewHandler(NewMockService(ctrl), pinger, slog.New(slog.DiscardHandler)).Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReady_CodeServiceUnavailable_DatabaseUnavailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	pinger := NewMockPinger(ctrl)
	pinger.EXPECT().Ping(gomock.Any()).Return(errors.New("connection refused"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	NewHandler(NewMockService(ctrl), pinger, slog.New(slog.DiscardHandler)).Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("статус: got %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var body api.HealthResponse
	decodeJSON(t, rec, &body)

	if body.Status != api.Unavailable {
		t.Errorf("статус в теле: got %q, want %q", body.Status, api.Unavailable)
	}
}
