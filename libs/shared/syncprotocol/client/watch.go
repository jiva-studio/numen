package client

import (
	"context"
	"errors"
	"net/http"
	"time"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

// WatchClient opens the stream of change hints of one server with the bearer of the session. The http.Client
// must have no Timeout: a Client.Timeout ends the read of the body, and so the stream.
type WatchClient struct {
	rpc syncv1connect.WatchServiceClient
}

// NewWatchClient returns the client of the server at address. token returns the session token and is read when
// the stream opens.
func NewWatchClient(address string, httpClient *http.Client, token func() string) *WatchClient {
	return &WatchClient{rpc: syncv1connect.NewWatchServiceClient(httpClient, address, newClientOptions(token))}
}

// WatchStream is an open stream of change hints whose Started message was read.
type WatchStream struct{}

// WatchChanges opens the stream and reads its first message.
func (c *WatchClient) WatchChanges(context.Context) (*WatchStream, error) {
	return nil, errors.New("not implemented")
}

// Heartbeat returns the longest silence the server announced.
func (s *WatchStream) Heartbeat() time.Duration { return 0 }

// Receive returns the next message, or io.EOF at a clean end.
func (s *WatchStream) Receive() (*syncv1.WatchChangesResponse, error) {
	return nil, errors.New("not implemented")
}

// Close ends the stream.
func (s *WatchStream) Close() error { return errors.New("not implemented") }
