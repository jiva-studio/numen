package fakeserver

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// watcher is one open stream: the latest seq of each vault that changed and was not sent yet, a signal of
// capacity one that wakes the stream, and a channel that ends it.
type watcher struct {
	account string
	pending map[string]uint64
	signal  chan struct{}
	end     chan struct{}
}

// streamSet holds the open streams. Its lock is taken after the lock of the server, never before.
type streamSet struct {
	mu        sync.Mutex
	heartbeat time.Duration
	watchers  map[*watcher]struct{}
}

func (s *streamSet) add(account string) *watcher {
	s.mu.Lock()
	defer s.mu.Unlock()
	w := &watcher{account: account, pending: map[string]uint64{}, signal: make(chan struct{}, 1), end: make(chan struct{})}
	s.watchers[w] = struct{}{}
	return w
}

func (s *streamSet) remove(w *watcher) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.watchers, w)
}

func (s *streamSet) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.watchers)
}

func (s *streamSet) endAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for w := range s.watchers {
		close(w.end)
	}
	clear(s.watchers)
}

func (s *streamSet) merge(w *watcher, seqs map[string]uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for vault, seq := range seqs {
		w.pending[vault] = max(w.pending[vault], seq)
	}
	wake(w)
}

func (s *streamSet) announce(account, vault string, seq uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for w := range s.watchers {
		if w.account != account {
			continue
		}
		w.pending[vault] = max(w.pending[vault], seq)
		wake(w)
	}
}

func (s *streamSet) take(w *watcher) map[string]uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	taken := maps.Clone(w.pending)
	clear(w.pending)
	return taken
}

func wake(w *watcher) {
	select {
	case w.signal <- struct{}{}:
	default:
	}
}

// EndStreams ends every open stream of change hints cleanly.
func (s *Server) EndStreams() { s.streams.endAll() }

// CountStreams returns how many streams of change hints are open.
func (s *Server) CountStreams() int { return s.streams.count() }

// WatchChanges streams the hints of the vaults of the account of the session: Started, the vaults that have a
// seq, then a hint for each commit and a heartbeat for each silence of the announced length.
func (s *Server) WatchChanges(ctx context.Context, req *connect.Request[syncv1.WatchChangesRequest], stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	session, err := s.authenticateSession(req.Header())
	if err != nil {
		return err
	}
	w := s.streams.add(session.AccountFingerprint)
	defer s.streams.remove(w)
	s.streams.merge(w, s.listVaultSeqs(session.AccountFingerprint))
	if err := stream.Send(newStartedMessage(s.streams.heartbeat)); err != nil {
		return fmt.Errorf("send started: %w", err)
	}
	return s.runStream(ctx, w, session.Token, stream)
}

func (s *Server) listVaultSeqs(account string) map[string]uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	seqs := map[string]uint64{}
	for id, vault := range s.vaults {
		if vault.account == account && vault.seq > 0 {
			seqs[id] = vault.seq
		}
	}
	return seqs
}

func (s *Server) runStream(ctx context.Context, w *watcher, token string, stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	idle := time.NewTimer(s.streams.heartbeat)
	defer idle.Stop()
	for {
		if s.sendPending(w, stream) {
			idle.Reset(s.streams.heartbeat)
		}
		select {
		case <-w.signal:
		case <-w.end:
			return nil
		case <-ctx.Done():
			return nil
		case <-idle.C:
			if _, isLive := s.findLiveSession(token); !isLive {
				return nil
			}
			if err := stream.Send(&syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Heartbeat{Heartbeat: &syncv1.Heartbeat{}}}); err != nil {
				return fmt.Errorf("send heartbeat: %w", err)
			}
			idle.Reset(s.streams.heartbeat)
		}
	}
}

func (s *Server) sendPending(w *watcher, stream *connect.ServerStream[syncv1.WatchChangesResponse]) bool {
	hints := s.streams.take(w)
	for _, vault := range slices.Sorted(maps.Keys(hints)) {
		hint := &syncv1.ChangeHint{VaultId: []byte(vault), Seq: hints[vault]}
		if stream.Send(&syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Hint{Hint: hint}}) != nil {
			return false
		}
	}
	return len(hints) > 0
}

func newStartedMessage(heartbeat time.Duration) *syncv1.WatchChangesResponse {
	seconds := uint32((heartbeat + time.Second - 1) / time.Second)
	return &syncv1.WatchChangesResponse{Event: &syncv1.WatchChangesResponse_Started{Started: &syncv1.Started{HeartbeatSeconds: seconds}}}
}

type callCounter struct{ server *Server }

func (c callCounter) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		c.server.countCall(req.Spec().Procedure)
		return next(ctx, req)
	}
}

func (callCounter) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (c callCounter) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		c.server.countCall(conn.Spec().Procedure)
		return next(ctx, conn)
	}
}

func (s *Server) countCall(procedure string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[procedure]++
}

func (s *Server) serveStreams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprintf(w, "%d\n", s.CountStreams())
}

func (s *Server) serveStreamsEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.EndStreams()
	w.WriteHeader(http.StatusNoContent)
}
