package httpsender

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/MishkaGql/template/api"
	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

func TestParseRequest(t *testing.T) {
	type requestBody struct {
		Price int `json:"price"`
	}

	tests := []struct {
		name     string
		body     string
		want     requestBody
		wantErr  bool
		wantCode errs.Code
	}{
		{
			name: "Success",
			body: `{"price":1450}`,

			want: requestBody{Price: 1450},
		},
		{
			name: "DecodeError_WrongType",
			body: `{"price":"1450"}`,

			wantErr: true,
		},
		{
			name: "InvalidRequest_MultipleValues",
			body: `{"price":1}{"price":2}`,

			wantErr:  true,
			wantCode: errs.InvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var got requestBody

			err := ParseRequest(req, &got)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRequest(%q) error = %v, wantErr %t", tt.body, err, tt.wantErr)
			}

			publicErr, ok := errs.AsError(err)
			if wantPublic := tt.wantCode != ""; ok != wantPublic {
				t.Fatalf("ParseRequest(%q) returns *errs.Error = %t, want %t (error: %v)", tt.body, ok, wantPublic, err)
			}

			if ok && publicErr.Code() != tt.wantCode {
				t.Errorf("ParseRequest(%q) code = %q, want %q", tt.body, publicErr.Code(), tt.wantCode)
			}

			if err == nil {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("ParseRequest(%q) mismatch (-want +got):\n%s", tt.body, diff)
				}
			}
		})
	}
}

func TestSendResponse(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		want := api.HealthResponse{Status: api.Ok}
		rec := httptest.NewRecorder()

		SendResponse(rec, MIMEJSON, http.StatusOK, want)

		if rec.Code != http.StatusOK {
			t.Errorf("SendResponse() status = %d, want 200", rec.Code)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("SendResponse() content type = %q, want application/json", got)
		}

		var got api.HealthResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode SendResponse() response: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("SendResponse() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("InternalError_MarshalFailed", func(t *testing.T) {
		t.Parallel()

		data := api.Coordinates{Latitude: math.NaN()}
		rec := httptest.NewRecorder()

		SendResponse(rec, MIMEJSON, http.StatusCreated, data)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("SendResponse() status = %d, want 500", rec.Code)
		}

		if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("SendResponse() content type = %q, want application/problem+json", got)
		}

		var got api.Problem
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode SendResponse() problem: %v", err)
		}

		detail := "Internal server error"
		want := api.Problem{
			Type:   "https://tripgo.example/problems/internal-error",
			Title:  "Internal Server Error",
			Status: 500,
			Code:   "internal_error",
			Detail: &detail,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("SendResponse() problem mismatch (-want +got):\n%s", diff)
		}
	})
}
