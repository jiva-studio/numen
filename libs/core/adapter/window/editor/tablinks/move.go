package tablinks

import "time"

// MoveTimeout is how long a move waits for the target window to accept the tab.
const MoveTimeout = 10 * time.Second

// MovedTab is a tab sent to another window. State is the JSON from the getState
// of its kind, empty when the kind saves none.
type MovedTab struct {
	ID, Kind, Path, Title, State string
}

// TabMove is a tab sent to the window WindowID.
type TabMove struct {
	WindowID string
	Tab      MovedTab
}

// isMovingInto reports whether a tab has a pending move into a window.
func (r *Registry) isMovingInto(windowID, tabID string) bool {
	pending, ok := r.pendingMoves[tabID]
	return ok && pending.targetWindowID == windowID
}

// Highlight turns the outline of the window WindowID on or off.
type Highlight struct {
	WindowID string
	IsOn     bool
}

// MoveError is why a move did not complete.
type MoveError int

const (
	// MoveErrorNone is a move the target window accepted.
	MoveErrorNone MoveError = iota
	// MoveErrorWindowClosed is a target window with no watch, or one whose
	// last watch ended before it accepted the tab.
	MoveErrorWindowClosed
	// MoveErrorTimeout is a target window that did not accept the tab within
	// MoveTimeout, or an Accept for a move no longer pending.
	MoveErrorTimeout
)

// move is one pending move of a tab.
type move struct {
	targetWindowID string
	result         chan MoveError
	cancel         func() bool
}

// Move sends a tab to another window and returns a channel that receives the
// result once. The source window owns the tab until the target window calls
// Accept. A newer move of the same tab ends the older one with
// MoveErrorTimeout.
func (r *Registry) Move(sourceWindowID, targetWindowID string, tab MovedTab) <-chan MoveError {
	result := make(chan MoveError, 1)
	r.mu.Lock()
	defer r.mu.Unlock()
	if targetWindowID == sourceWindowID || len(r.watches[targetWindowID]) == 0 {
		result <- MoveErrorWindowClosed
		return result
	}
	if previous, ok := r.pendingMoves[tab.ID]; ok {
		r.endMove(tab.ID, previous, MoveErrorTimeout)
	}
	pending := &move{targetWindowID: targetWindowID, result: result}
	r.pendingMoves[tab.ID] = pending
	pending.cancel = r.schedule(MoveTimeout, func() { r.expireMove(tab.ID, pending) })
	r.events.Publish(Event{Move: &TabMove{WindowID: targetWindowID, Tab: tab}})
	return result
}

// Accept completes the move of a tab into a window. A known tab changes owner
// to that window and keeps its group.
func (r *Registry) Accept(windowID, tabID string) MoveError {
	r.mu.Lock()
	defer r.mu.Unlock()
	pending, ok := r.pendingMoves[tabID]
	if !ok || pending.targetWindowID != windowID {
		return MoveErrorTimeout
	}
	if tab, isKnown := r.tabs[tabID]; isKnown {
		r.setOwner(windowID, tab)
		r.windows[windowID] = appendMissing(r.windows[windowID], tabID)
		r.publishSnapshot()
	}
	r.endMove(tabID, pending, MoveErrorNone)
	return MoveErrorNone
}

// Highlight turns the outline of a window on or off. A window with no watch
// is ignored.
func (r *Registry) Highlight(windowID string, isOn bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.watches[windowID]) == 0 {
		return
	}
	r.events.Publish(Event{Highlight: &Highlight{WindowID: windowID, IsOn: isOn}})
}

// expireMove ends a move the target window did not accept in time. An expiry
// of a move that already ended does nothing.
func (r *Registry) expireMove(tabID string, pending *move) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingMoves[tabID] == pending {
		r.endMove(tabID, pending, MoveErrorTimeout)
	}
}

// endMovesInto ends every move into a window that stopped watching.
func (r *Registry) endMovesInto(windowID string) {
	for tabID, pending := range r.pendingMoves {
		if pending.targetWindowID == windowID {
			r.endMove(tabID, pending, MoveErrorWindowClosed)
		}
	}
}

func (r *Registry) endMove(tabID string, pending *move, outcome MoveError) {
	pending.cancel()
	delete(r.pendingMoves, tabID)
	pending.result <- outcome
}
