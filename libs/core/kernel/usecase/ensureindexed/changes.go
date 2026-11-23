package ensureindexed

import "sync"

// Changes signals when indexed vault data changes. The zero value is ready to use.
//
// A busy listener is skipped: every message carries the same signal, and a
// dropped message is covered by the next one.
type Changes struct {
	mu          sync.Mutex
	next        int
	subscribers map[int]chan struct{}
}

func (f *Changes) Subscribe() (<-chan struct{}, func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subscribers == nil {
		f.subscribers = make(map[int]chan struct{})
	}
	line := make(chan struct{}, 1)
	pos := f.next
	f.next++
	f.subscribers[pos] = line
	return line, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		if target, is := f.subscribers[pos]; is {
			delete(f.subscribers, pos)
			close(target)
		}
	}
}

func (f *Changes) ReportChange() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range f.subscribers {
		select {
		case line <- struct{}{}:
		default:
		}
	}
}
