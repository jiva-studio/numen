package fakeserver

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	"connectrpc.com/connect"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver/internal/streams"
)

// EndStreams ends every open stream of change hints cleanly.
func (s *Server) EndStreams() { s.streams.EndAll() }

// CountStreams returns how many streams of change hints are open.
func (s *Server) CountStreams() int { return s.streams.Count() }

// WatchChanges streams the hints of the vaults of the account of the session: Started, the vaults that have a
// seq, then a hint for each commit and a heartbeat for each silence of the announced length.
func (s *Server) WatchChanges(ctx context.Context, req *connect.Request[syncv1.WatchChangesRequest], stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	session, err := s.authenticateSession(req.Header())
	if err != nil {
		return err
	}
	w := s.streams.Add(session.AccountFingerprint)
	defer s.streams.Remove(w)
	s.streams.Merge(w, s.listVaultSeqs(session.AccountFingerprint))
	if err := stream.Send(newStartedMessage(s.streams.Heartbeat)); err != nil {
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

func (s *Server) runStream(ctx context.Context, w *streams.Watcher, token string, stream *connect.ServerStream[syncv1.WatchChangesResponse]) error {
	idle := time.NewTimer(s.streams.Heartbeat)
	defer idle.Stop()
	for {
		if s.sendPending(w, stream) {
			idle.Reset(s.streams.Heartbeat)
		}
		select {
		case <-w.Signal:
		case <-w.End:
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
			idle.Reset(s.streams.Heartbeat)
		}
	}
}

func (s *Server) sendPending(w *streams.Watcher, stream *connect.ServerStream[syncv1.WatchChangesResponse]) bool {
	hints := s.streams.Take(w)
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
	if _, err := fmt.Fprintf(w, "%d\n", s.CountStreams()); err != nil {
		return
	}
}

func (s *Server) serveStreamsEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.EndStreams()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) servePulls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	counts := maps.Clone(s.pulls)
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(counts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
