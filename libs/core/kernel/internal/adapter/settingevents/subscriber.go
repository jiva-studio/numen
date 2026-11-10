package settingevents

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// subscriber is one reader. Its batches wait in a queue, so a reader that is slow to take them
// never holds up the write that made them.
type subscriber struct {
	out   chan []settings.Setting
	leave chan struct{}
	wake  chan struct{}
	once  sync.Once

	mu    sync.Mutex
	queue [][]settings.Setting
}

func newSubscriber() *subscriber {
	return &subscriber{
		out:   make(chan []settings.Setting),
		leave: make(chan struct{}),
		wake:  make(chan struct{}, 1),
	}
}

func (s *subscriber) push(batch []settings.Setting) {
	s.mu.Lock()
	s.queue = append(s.queue, batch)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *subscriber) next() ([]settings.Setting, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.queue) == 0 {
		return nil, false
	}
	batch := s.queue[0]
	s.queue = s.queue[1:]
	return batch, true
}

func (s *subscriber) deliver() {
	defer close(s.out)
	for {
		batch, found := s.next()
		if !found {
			select {
			case <-s.wake:
				continue
			case <-s.leave:
				return
			}
		}
		select {
		case s.out <- batch:
		case <-s.leave:
			return
		}
	}
}
