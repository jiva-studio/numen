package editor

import (
	"context"
	"sync/atomic"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
)

type inflight struct {
	in broadcast.Inflight
}

func (w *inflight) begin() bool           { return w.in.Begin() }
func (w *inflight) finish()               { w.in.Finish() }
func (w *inflight) seal() <-chan struct{} { return w.in.Seal() }

// Gate is the door on every question of the window. It counts the questions being answered and refuses
// new ones once the window is shut.
type Gate struct {
	closed    atomic.Bool
	questions inflight
}

// Close rejects all new requests permanently. It returns after the requests
// already accepted have completed, so the index and embedder stay available
// for their whole duration.
func (g *Gate) Close() {
	g.closed.Store(true)
	<-g.questions.seal()
}

func (g *Gate) isClosed() bool { return g.closed.Load() }

// CountQuestions counts each call to a service among the questions the window
// is answering, and refuses it once the window is shut.
func (g *Gate) CountQuestions() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !g.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, errShut)
				}
				defer g.questions.finish()
				return next(ctx, r)
			}
		},
	))
}
