package wire

import (
	"context"
	"errors"
	"sync/atomic"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/broadcast"
)

// ErrShut is returned for a request to a window that has been shut; the window answers no further requests.
var ErrShut = errors.New("this window is shut")

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

// IsClosed reports whether the window is shut.
func (g *Gate) IsClosed() bool { return g.closed.Load() }

// BeginQuestion counts a question being answered, and reports false once the window is shut.
func (g *Gate) BeginQuestion() bool { return g.questions.begin() }

// FinishQuestion ends a question that BeginQuestion counted.
func (g *Gate) FinishQuestion() { g.questions.finish() }

// CountQuestions counts each call to a service among the questions the window
// is answering, and refuses it once the window is shut.
func (g *Gate) CountQuestions() connect.HandlerOption {
	return connect.WithInterceptors(connect.UnaryInterceptorFunc(
		func(next connect.UnaryFunc) connect.UnaryFunc {
			return func(ctx context.Context, r connect.AnyRequest) (connect.AnyResponse, error) {
				if !g.questions.begin() {
					return nil, connect.NewError(connect.CodeUnavailable, ErrShut)
				}
				defer g.questions.finish()
				return next(ctx, r)
			}
		},
	))
}
