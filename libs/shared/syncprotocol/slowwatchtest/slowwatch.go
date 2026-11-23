// Package slowwatchtest is a watch service whose stream is slow, for the tests of a client that must keep a
// stream open longer than the timeout of its request client.
package slowwatchtest

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type slowWatch struct {
	syncv1connect.UnimplementedWatchServiceHandler

	delay time.Duration
}

// NewHandler returns a handler of the watch service whose stream sends Started at once and one hint at seq 4
// of vault "vault-of-16-byte" after delay.
func NewHandler(delay time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewWatchServiceHandler(slowWatch{UnimplementedWatchServiceHandler: syncv1connect.UnimplementedWatchServiceHandler{}, delay: delay}))
	return mux
}

func (s slowWatch) WatchChanges(ctx context.Context, _ *connect.Request[syncv1.WatchChangesRequest], stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	started := &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Started{Started: &syncv1.Started{HeartbeatSeconds: 1}}}
	if err := stream.Send(started); err != nil {
		return err
	}
	select {
	case <-time.After(s.delay):
	case <-ctx.Done():
		return ctx.Err()
	}
	hint := &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Hint{Hint: &syncv1.ChangeHint{VaultId: []byte("vault-of-16-byte"), Seq: 4}}}
	return stream.Send(hint)
}
