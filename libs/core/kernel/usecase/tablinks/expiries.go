package tablinks

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/port/tablinkport"
)

// MoveTimeout is how long a move waits for the target window to accept the tab.
const MoveTimeout = 10 * time.Second

// expiries are the timers of the state: the removal of a window, a move, a detach and the memory of an
// accepted move. Each run takes the lock of the state and does nothing when its entry has been replaced.
type expiries struct {
	state    *state
	grace    time.Duration
	schedule tablinkport.Schedule
}

func (e *expiries) scheduleWindowRemoval(windowID string) {
	pending := &expiry{}
	e.state.presence.removals[windowID] = pending
	pending.cancel = e.schedule(e.grace, func() { e.expireWindow(windowID, pending) })
}

func (e *expiries) cancelWindowRemoval(windowID string) {
	if pending, ok := e.state.presence.removals[windowID]; ok {
		pending.cancel()
		delete(e.state.presence.removals, windowID)
	}
}

func (e *expiries) expireWindow(windowID string, pending *expiry) {
	s := e.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.presence.removals[windowID] != pending {
		return
	}
	delete(s.presence.removals, windowID)
	s.removeWindow(windowID)
}

// keepAcceptingWindow keeps the tabs of a window that has no watch until its grace period ends.
func (e *expiries) keepAcceptingWindow(windowID string) {
	p := e.state.presence
	if p.hasWatch(windowID) {
		return
	}
	pending, ok := p.removals[windowID]
	if !ok {
		e.scheduleWindowRemoval(windowID)
		pending = p.removals[windowID]
	}
	pending.hasAccepted = true
}

func (e *expiries) scheduleMove(tabID string, pending *move) {
	pending.cancel = e.schedule(MoveTimeout, func() { e.expireMove(tabID, pending) })
}

func (e *expiries) expireMove(tabID string, pending *move) {
	s := e.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.transit.moves[tabID] == pending {
		s.transit.endMove(tabID, pending, tablink.MoveErrorTimeout)
	}
}

func (e *expiries) scheduleDetach(tabID string, pending *detach) {
	pending.cancel = e.schedule(e.grace, func() { e.expireDetach(tabID, pending) })
}

func (e *expiries) expireDetach(tabID string, pending *detach) {
	s := e.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.transit.detaches[tabID] != pending {
		return
	}
	s.transit.endDetach(tabID, pending, tablink.MoveErrorTimeout)
	s.dropUnreportedTab(tabID)
}

func (e *expiries) recordAccepted(moveID, windowID string) {
	e.state.transit.accepted[moveID] = windowID
	e.schedule(e.grace, func() { e.forgetAccepted(moveID) })
}

func (e *expiries) forgetAccepted(moveID string) {
	e.state.mu.Lock()
	defer e.state.mu.Unlock()
	delete(e.state.transit.accepted, moveID)
}
