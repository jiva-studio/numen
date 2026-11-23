package client_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
)

type watchStub struct {
	syncv1connect.UnimplementedWatchServiceHandler

	mu       sync.Mutex
	headers  []http.Header
	fail     error
	messages []*syncv1.WatchChangesResponse
}

func (s *watchStub) WatchChanges(_ context.Context, req *connect.Request[syncv1.WatchChangesRequest], stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	s.mu.Lock()
	s.headers = append(s.headers, req.Header().Clone())
	fail, messages := s.fail, s.messages
	s.mu.Unlock()
	if fail != nil {
		return fail
	}
	for _, message := range messages {
		if err := stream.Send(message); err != nil {
			return err
		}
	}
	return nil
}

func serveWatchStub(tb testing.TB, serverVersion uint32, stub *watchStub) string {
	tb.Helper()
	mux := http.NewServeMux()
	mux.Handle(syncv1connect.NewWatchServiceHandler(stub, connect.WithInterceptors(syncprotocol.NewVersionHandlerInterceptor(serverVersion))))
	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)
	return server.URL
}

func startedMessage(seconds uint32) *syncv1.WatchChangesResponse {
	return &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Started{Started: &syncv1.Started{HeartbeatSeconds: seconds}}}
}

func hintMessage(vault []byte, seq uint64) *syncv1.WatchChangesResponse {
	return &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Hint{Hint: &syncv1.ChangeHint{VaultId: vault, Seq: seq}}}
}

func newWatchClient(address string, token func() string) *syncclient.WatchClient {
	return syncclient.NewWatchClient(address, &http.Client{}, token)
}

func TestWatchChanges_ReadsStartedAndReturnsTheAnnouncedHeartbeat(t *testing.T) {
	t.Parallel()
	address := serveWatchStub(t, syncprotocol.Version, &watchStub{messages: []*syncv1.WatchChangesResponse{startedMessage(7)}})

	stream, err := newWatchClient(address, func() string { return "token-of-the-session" }).WatchChanges(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()

	if got := stream.Heartbeat(); got != 7*time.Second {
		t.Errorf("heartbeat = %v, want 7s", got)
	}
}

func TestWatchChanges_ReturnsTheMessagesAfterStartedInOrderAndEOFAtACleanEnd(t *testing.T) {
	t.Parallel()
	vault := []byte("vault-of-16-byte")
	address := serveWatchStub(t, syncprotocol.Version, &watchStub{messages: []*syncv1.WatchChangesResponse{startedMessage(1), hintMessage(vault, 3), hintMessage(vault, 4)}})
	stream, err := newWatchClient(address, func() string { return "t" }).WatchChanges(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()

	for _, want := range []uint64{3, 4} {
		message, err := stream.Receive()
		if err != nil {
			t.Fatal(err)
		}
		if got := message.GetHint().GetSeq(); got != want {
			t.Errorf("seq = %d, want %d", got, want)
		}
	}
	if _, err := stream.Receive(); !errors.Is(err, io.EOF) {
		t.Errorf("error at the end = %v, want io.EOF", err)
	}
}

func TestWatchChanges_SendsTheVersionAndTheBearerOnceWhenItOpens(t *testing.T) {
	t.Parallel()
	stub := &watchStub{messages: []*syncv1.WatchChangesResponse{startedMessage(1)}}
	address := serveWatchStub(t, syncprotocol.Version, stub)

	stream, err := newWatchClient(address, func() string { return "token-of-the-session" }).WatchChanges(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}()

	stub.mu.Lock()
	defer stub.mu.Unlock()
	if len(stub.headers) != 1 {
		t.Fatalf("opens = %d, want 1", len(stub.headers))
	}
	if got := stub.headers[0].Get("Authorization"); got != "Bearer token-of-the-session" {
		t.Errorf("Authorization = %q, want the bearer of the token", got)
	}
}

func TestWatchChanges_FailsWhenTheFirstMessageIsNotStarted(t *testing.T) {
	t.Parallel()
	address := serveWatchStub(t, syncprotocol.Version, &watchStub{messages: []*syncv1.WatchChangesResponse{hintMessage([]byte("vault-of-16-byte"), 1)}})

	_, err := newWatchClient(address, func() string { return "t" }).WatchChanges(t.Context())

	if err == nil || err.Error() != "the sync server did not start the stream" {
		t.Errorf("error = %v, want the sync server did not start the stream", err)
	}
}

func TestWatchChanges_MapsAnErrorOfTheOpenToTheErrorsOfThePackage(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		fail error
		want error
	}{
		"no live session": {fail: connect.NewError(connect.CodeUnauthenticated, errors.New("no session")), want: syncprotocol.ErrUnauthenticated},
		"server is away":  {fail: connect.NewError(connect.CodeUnavailable, errors.New("away")), want: syncprotocol.ErrUnreachable},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			address := serveWatchStub(t, syncprotocol.Version, &watchStub{fail: test.fail})

			_, err := newWatchClient(address, func() string { return "t" }).WatchChanges(t.Context())

			if !errors.Is(err, test.want) {
				t.Errorf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestWatchChanges_KeepsResourceExhaustedAsAConnectError(t *testing.T) {
	t.Parallel()
	address := serveWatchStub(t, syncprotocol.Version, &watchStub{fail: connect.NewError(connect.CodeResourceExhausted, errors.New("too many streams"))})

	_, err := newWatchClient(address, func() string { return "t" }).WatchChanges(t.Context())

	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Errorf("code = %v, want resource exhausted", connect.CodeOf(err))
	}
}

func TestWatchChanges_ReportsAServerOfAnotherVersionAsUpdateRequired(t *testing.T) {
	t.Parallel()
	address := serveWatchStub(t, syncprotocol.Version+1, &watchStub{messages: []*syncv1.WatchChangesResponse{startedMessage(1)}})

	_, err := newWatchClient(address, func() string { return "t" }).WatchChanges(t.Context())

	if !errors.Is(err, syncprotocol.ErrUpdateRequired) {
		t.Errorf("error = %v, want ErrUpdateRequired", err)
	}
}

func TestWatchChanges_ReturnsUnreachableForAServerThatIsNotThere(t *testing.T) {
	t.Parallel()
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closed.Close()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	_, err := newWatchClient(closed.URL, func() string { return "t" }).WatchChanges(ctx)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
}
