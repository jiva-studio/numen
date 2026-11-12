package windowsession

import (
	"sync"
	"sync/atomic"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// TabReports keeps the tabs the window reported last. The zero value holds no report.
type TabReports struct {
	open   atomic.Pointer[windowdomain.OpenTabs]
	signal reportSignal
}

// ReportOpenTabs keeps the tabs the window reported and wakes whoever waits for a report.
func (r *TabReports) ReportOpenTabs(open windowdomain.OpenTabs) {
	r.open.Store(&open)
	r.signal.notify()
}

// GetOpenTabs returns the open tabs as last reported by the window, or the empty set when the window
// has not reported any.
func (r *TabReports) GetOpenTabs() windowdomain.OpenTabs {
	if open := r.open.Load(); open != nil {
		return *open
	}
	return windowdomain.OpenTabs{}
}

// reportSignal closes a channel when a report arrives. The zero value is ready to use.
type reportSignal struct {
	mu   sync.Mutex
	next chan struct{}
}

func (s *reportSignal) wait() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next == nil {
		s.next = make(chan struct{})
	}
	return s.next
}

func (s *reportSignal) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.next != nil {
		close(s.next)
		s.next = nil
	}
}
