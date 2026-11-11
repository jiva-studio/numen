package tablinks

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// MoveTimeout is how long a move waits for the target window to accept the tab.
const MoveTimeout = 10 * time.Second

// expiries are the timers of the state: the removal of a window, a move, a detach and the memory of an
// accepted move. Each run takes the lock of the state and does nothing when its entry has been replaced.
type expiries struct {
	grace    time.Duration
	schedule Schedule
}

func (e *expiries) scheduleWindowRemoval(s *state, windowID string) {
	pending := &expiry{}
	s.roster.ensure(windowID).removal = pending
	pending.cancel = e.schedule(e.grace, func() { e.expireWindow(s, windowID, pending) })
}

func (e *expiries) cancelWindowRemoval(s *state, windowID string) {
	if entry := s.roster.windows[windowID]; entry != nil && entry.removal != nil {
		entry.removal.cancel()
		entry.removal = nil
		s.roster.prune(windowID)
	}
}

func (e *expiries) expireWindow(s *state, windowID string, pending *expiry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.roster.windows[windowID]
	if entry == nil || entry.removal != pending {
		return
	}
	entry.removal = nil
	s.removeWindow(windowID)
	s.roster.prune(windowID)
}

func (e *expiries) keepAcceptingWindow(s *state, windowID string) {
	if s.roster.windows[windowID].hasWatch() {
		return
	}
	entry := s.roster.ensure(windowID)
	if entry.removal == nil {
		e.scheduleWindowRemoval(s, windowID)
	}
	entry.removal.hasAccepted = true
}

func (e *expiries) scheduleMove(s *state, tabID string, pending *move) {
	pending.cancel = e.schedule(MoveTimeout, func() { e.expireMove(s, tabID, pending) })
}

func (e *expiries) expireMove(s *state, tabID string, pending *move) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, _ := s.transit.getMove(tabID); current == pending {
		s.transit.endMove(tabID, pending, tablink.MoveErrorTimeout)
	}
}

func (e *expiries) scheduleDetach(s *state, tabID string, pending *detach) {
	pending.cancel = e.schedule(e.grace, func() { e.expireDetach(s, tabID, pending) })
}

func (e *expiries) expireDetach(s *state, tabID string, pending *detach) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, _ := s.transit.getDetach(tabID); current != pending {
		return
	}
	s.transit.endDetach(tabID, pending, tablink.MoveErrorTimeout)
	s.dropUnreportedTab(tabID)
}

func (e *expiries) recordAccepted(s *state, moveID, windowID string) {
	s.transit.accepted[moveID] = windowID
	e.schedule(e.grace, func() { e.forgetAccepted(s, moveID) })
}

func (e *expiries) forgetAccepted(s *state, moveID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.transit.accepted, moveID)
}
