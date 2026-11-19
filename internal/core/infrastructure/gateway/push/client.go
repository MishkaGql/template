package push

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/MishkaGql/template/api/pushclient"
)

type Client struct {
	api       *pushclient.Client
	transport *http.Transport
}

func New(url string, timeout time.Duration) (*Client, error) {
	transport := &http.Transport{IdleConnTimeout: timeout, ResponseHeaderTimeout: timeout}

	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(transport),
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	api, err := pushclient.NewClient(url, pushclient.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &Client{api: api, transport: transport}, nil
}

func (c *Client) RequestPosition(ctx context.Context, tripID, driverID uuid.UUID) (err error) {
	data := map[string]string{"trip_id": tripID.String()}

	response, err := c.api.SendPush(ctx, pushclient.PushRequest{
		RequestId:   uuid.NewString(),
		Kind:        pushclient.REQUESTPOSITION,
		RecipientId: driverID,
		Data:        &data,
	})
	if err != nil {
		return fmt.Errorf("send position request: %w", err)
	}

	defer func() {
		err = errors.Join(err, response.Body.Close())
	}()

	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("push service returned HTTP %d", response.StatusCode)
	}
	_, err = io.Copy(io.Discard, response.Body)
	return err
}

func (c *Client) Close() {
	c.transport.CloseIdleConnections()
}
