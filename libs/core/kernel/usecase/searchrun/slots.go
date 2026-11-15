package searchrun

import (
	"context"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// slots counts the running searches. A search waiting for a slot is queued by its priority, interactive
// before background, each in the order of arrival.
type slots struct {
	mu          sync.Mutex
	free        int
	interactive []chan struct{}
	background  []chan struct{}
}

func newSlots(count int) *slots {
	return &slots{free: count}
}

func (s *slots) acquire(ctx context.Context, priority modelcall.Priority) error {
	s.mu.Lock()
	if s.free > 0 && len(s.interactive)+len(s.background) == 0 {
		s.free--
		s.mu.Unlock()
		return nil
	}
	granted := make(chan struct{})
	if priority == modelcall.PriorityInteractive {
		s.interactive = append(s.interactive, granted)
	} else {
		s.background = append(s.background, granted)
	}
	s.mu.Unlock()
	select {
	case <-granted:
		return nil
	case <-ctx.Done():
		s.leave(granted)
		return ctx.Err()
	}
}

func (s *slots) leave(granted chan struct{}) {
	s.mu.Lock()
	before := len(s.interactive) + len(s.background)
	s.interactive = slices.DeleteFunc(s.interactive, func(c chan struct{}) bool { return c == granted })
	s.background = slices.DeleteFunc(s.background, func(c chan struct{}) bool { return c == granted })
	isQueued := len(s.interactive)+len(s.background) < before
	s.mu.Unlock()
	if !isQueued {
		s.release()
	}
}

func (s *slots) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case len(s.interactive) > 0:
		close(s.interactive[0])
		s.interactive = s.interactive[1:]
	case len(s.background) > 0:
		close(s.background[0])
		s.background = s.background[1:]
	default:
		s.free++
	}
}
