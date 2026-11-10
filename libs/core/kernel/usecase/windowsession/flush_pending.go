package windowsession

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// FlushPending has every page write what only it holds, then waits for the writes already taken.
type FlushPending struct {
	life *lifecycle
}

// Execute flushes the pages. Calling it again returns true immediately.
// It returns false, with no write discarded, when a page holds work that needs
// user confirmation. A close that arrives while a flush runs returns false: the
// window stays, and the next close flushes again.
func (u FlushPending) Execute(ctx context.Context) bool {
	rt, deps := u.life.rt, u.life.deps
	switch err := rt.guard.tryBegin(); {
	case errors.Is(err, window.ErrClosing):
		return true
	case err != nil:
		return false
	}

	flushed := flushPending(ctx, deps.Window.Pages, deps.Window.Writes)
	rt.guard.end(flushed)
	return flushed
}
