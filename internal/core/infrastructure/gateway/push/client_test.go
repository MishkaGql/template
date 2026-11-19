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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MishkaGql/template/api/pushclient"
)

func TestRequestPosition_Success(t *testing.T) {
	tripID := uuid.New()
	driverID := uuid.New()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/push", r.URL.Path)

		var payload pushclient.PushRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode push request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		assert.NotEmpty(t, payload.RequestId)
		assert.Equal(t, pushclient.REQUESTPOSITION, payload.Kind)
		assert.Equal(t, driverID, payload.RecipientId)
		assert.Equal(t, &map[string]string{"trip_id": tripID.String()}, payload.Data)

		w.WriteHeader(http.StatusAccepted)
	})

	err := client.RequestPosition(t.Context(), tripID, driverID)
	require.NoError(t, err)
}

func TestRequestPosition_HTTPError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			})

			err := client.RequestPosition(t.Context(), uuid.New(), uuid.New())
			require.ErrorContains(t, err, strconv.Itoa(status))
		})
	}
}

func TestRequestPosition_Cancelled(t *testing.T) {
	client, err := New("http://unused.invalid", time.Second)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = client.RequestPosition(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, context.Canceled)

	var transportErr *url.Error
	require.ErrorAs(t, err, &transportErr)
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(server.URL, time.Second)
	require.NoError(t, err)
	t.Cleanup(client.Close)

	return client
}
