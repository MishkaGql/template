package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/MishkaGql/template/pushclient"
)

func TestRequestPosition(t *testing.T) {
	tripID, driverID := uuid.New(), uuid.New()
	for _, status := range []int{202, 400, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/api/v1/push" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var req pushclient.PushRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				if req.Kind != pushclient.REQUESTPOSITION || req.RequestId == "" || req.RecipientId != driverID || req.Data == nil || (*req.Data)["trip_id"] != tripID.String() {
					t.Errorf("unexpected push payload %+v", req)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			c, err := New(server.URL, time.Second)
			require.NoError(t, err)
			defer c.Close()
			err = c.RequestPosition(t.Context(), tripID, driverID)
			if status == 202 {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, strconv.Itoa(status))
			}
		})
	}
	c, err := New("http://unused.invalid", time.Second)
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err = c.RequestPosition(ctx, tripID, driverID)
	require.ErrorIs(t, err, context.Canceled)
	var transportErr *url.Error
	require.ErrorAs(t, err, &transportErr)
}
