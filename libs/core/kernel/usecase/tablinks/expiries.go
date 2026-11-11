package tablinks

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// MoveTimeout is how long a move waits for the target window to accept the tab.
const MoveTimeout = 10 * time.Second

// expiryTimers are the timers of the state.
type expiryTimers interface {
	windowTimers
	transferTimers
}

// windowTimers are the timers that remove a window nobody watches.
type windowTimers interface {
	scheduleWindowRemoval(windowID string)
	cancelWindowRemoval(windowID string)
	keepAcceptingWindow(windowID string)
}

// transferTimers are the timers of the moves and detaches, and of the memory of an accepted move.
type transferTimers interface {
	scheduleMove(tabID string, pending *move)
	scheduleDetach(tabID string, pending *detach)
	recordAccepted(moveID, windowID string)
}

// expiries are the timers of the state: the removal of a window, a move, a detach and the memory of an
// accepted move. Each run takes the lock of the state and does nothing when its entry has been replaced.
type expiries struct {
	host     shared
	grace    time.Duration
	schedule Schedule
}

func (e *expiries) scheduleWindowRemoval(windowID string) {
	pending := &expiry{}
	e.host.peek().roster.presence.setRemoval(windowID, pending)
	pending.cancel = e.schedule(e.grace, func() { e.expireWindow(windowID, pending) })
}

func (e *expiries) cancelWindowRemoval(windowID string) {
	presence := e.host.peek().roster.presence
	if pending, ok := presence.getRemoval(windowID); ok {
		pending.cancel()
		presence.deleteRemoval(windowID)
	}
}

func (e *expiries) expireWindow(windowID string, pending *expiry) {
	s, release := e.host.hold()
	defer release()
	if current, _ := s.roster.presence.getRemoval(windowID); current != pending {
		return
	}
	s.roster.presence.deleteRemoval(windowID)
	s.removeWindow(windowID)
}

func (e *expiries) keepAcceptingWindow(windowID string) {
	p := e.host.peek().roster.presence
	if p.hasWatch(windowID) {
		return
	}
	pending, ok := p.getRemoval(windowID)
	if !ok {
		e.scheduleWindowRemoval(windowID)
		pending, _ = p.getRemoval(windowID)
	}
	pending.hasAccepted = true
}

func (e *expiries) scheduleMove(tabID string, pending *move) {
	pending.cancel = e.schedule(MoveTimeout, func() { e.expireMove(tabID, pending) })
}

func (e *expiries) expireMove(tabID string, pending *move) {
	s, release := e.host.hold()
	defer release()
	if current, _ := s.transit.getMove(tabID); current == pending {
		s.transit.endMove(tabID, pending, tablink.MoveErrorTimeout)
	}
}

func (e *expiries) scheduleDetach(tabID string, pending *detach) {
	pending.cancel = e.schedule(e.grace, func() { e.expireDetach(tabID, pending) })
}

func (e *expiries) expireDetach(tabID string, pending *detach) {
	s, release := e.host.hold()
	defer release()
	if current, _ := s.transit.getDetach(tabID); current != pending {
		return
	}
	s.transit.endDetach(tabID, pending, tablink.MoveErrorTimeout)
	s.dropUnreportedTab(tabID)
}

func (e *expiries) recordAccepted(moveID, windowID string) {
	e.host.peek().transit.putAccepted(moveID, windowID)
	e.schedule(e.grace, func() { e.forgetAccepted(moveID) })
}

func (e *expiries) forgetAccepted(moveID string) {
	s, release := e.host.hold()
	defer release()
	s.transit.deleteAccepted(moveID)
}
