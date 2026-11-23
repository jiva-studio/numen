package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

var errNotStarted = errors.New("the sync server did not start the stream")

// WatchClient opens the stream of change hints of one server with the bearer of the session. The http.Client
// must have no Timeout: a Client.Timeout ends the read of the body, and so the stream.
type WatchClient struct {
	rpc syncv1connect.WatchServiceClient
}

// NewWatchClient returns the client of the server at address. token returns the session token and is read when
// the stream opens.
func NewWatchClient(address string, httpClient *http.Client, token func() string) *WatchClient {
	return &WatchClient{rpc: syncv1connect.NewWatchServiceClient(httpClient, address, newClientOptions(address, token))}
}

// WatchStream is an open stream of change hints whose Started message was read.
type WatchStream struct {
	stream    *connect.ServerStreamForClient[syncv1.WatchChangesResponse]
	cancel    context.CancelFunc
	heartbeat time.Duration
}

// WatchChanges opens the stream and reads its first message, which must be Started. Connect reports the refusal
// of an open at the first read, so the refusal is returned here, mapped by syncprotocol.ReadError.
func (c *WatchClient) WatchChanges(ctx context.Context) (*WatchStream, error) {
	ctx, cancel := context.WithCancel(ctx)
	stream, err := c.rpc.WatchChanges(ctx, connect.NewRequest(&syncv1.WatchChangesRequest{}))
	if err != nil {
		cancel()
		return nil, syncprotocol.ReadError(err)
	}
	opened := &WatchStream{stream: stream, cancel: cancel}
	first, err := opened.Receive()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.Join(err, opened.Close())
	}
	if first.GetStarted() == nil {
		return nil, errors.Join(errNotStarted, opened.Close())
	}
	opened.heartbeat = time.Duration(first.GetStarted().GetHeartbeatSeconds()) * time.Second
	return opened, nil
}

// Heartbeat returns the longest silence the server announced.
func (s *WatchStream) Heartbeat() time.Duration { return s.heartbeat }

// Receive returns the next message, or io.EOF at a clean end.
func (s *WatchStream) Receive() (*syncv1.WatchChangesResponse, error) {
	if s.stream.Receive() {
		return s.stream.Msg(), nil
	}
	if err := s.stream.Err(); err != nil {
		return nil, syncprotocol.ReadError(err)
	}
	return nil, io.EOF
}

// Close ends the stream. The context ends first: Connect drains the body before it closes it, and a stream
// that is still open has no end to drain.
func (s *WatchStream) Close() error {
	s.cancel()
	return s.stream.Close()
}
