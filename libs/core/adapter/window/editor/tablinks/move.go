package tablinks

import (
	"strconv"
	"time"
)

// MoveTimeout is how long a move waits for the target window to accept the tab.
const MoveTimeout = 10 * time.Second

// SavedTab is a tab with the state its kind saved, as the wire Tab. State is
// the JSON from the getState of its kind, empty when the kind saves none.
type SavedTab struct {
	ID, Kind, Path, Title, State string
}

// MovedTab is a tab sent to the window WindowID, as the wire MovedTab. The
// window passes MoveID to Accept.
type MovedTab struct {
	WindowID string
	MoveID   string
	Tab      SavedTab
}

func (r *Registry) isMovingInto(windowID, tabID string) bool {
	pending, ok := r.pendingMoves[tabID]
	return ok && pending.targetWindowID == windowID
}

func (r *Registry) isDetachingInto(windowID, tabID string) bool {
	pending, ok := r.pendingDetaches[tabID]
	return ok && pending.windowID == windowID
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
	// MoveErrorNotOwner is a source window that does not own the tab.
	MoveErrorNotOwner
	// MoveErrorCancelled is a move that a newer move or detach of the tab, a
	// cancel, or another window that took the tab ended.
	MoveErrorCancelled
)

// move is one pending move of a tab.
type move struct {
	id             string
	targetWindowID string
	result         chan MoveError
	cancel         func() bool
}

// Move sends a tab to another window and returns a channel that receives the
// result once. The source window owns the tab until the target window calls
// Accept. It ends a pending move or detach of the same tab with
// MoveErrorCancelled. A known tab moves only from the window that owns it.
func (r *Registry) Move(sourceWindowID, targetWindowID string, tab SavedTab) <-chan MoveError {
	result := make(chan MoveError, 1)
	r.mu.Lock()
	defer r.mu.Unlock()
	if targetWindowID == sourceWindowID || len(r.watches[targetWindowID]) == 0 {
		result <- MoveErrorWindowClosed
		return result
	}
	if known, ok := r.tabs[tab.ID]; ok && known.WindowID != sourceWindowID {
		result <- MoveErrorNotOwner
		return result
	}
	r.endTransfers(tab.ID, MoveErrorCancelled)
	pending := &move{id: r.createMoveID(), targetWindowID: targetWindowID, result: result}
	r.pendingMoves[tab.ID] = pending
	pending.cancel = r.schedule(MoveTimeout, func() { r.expireMove(tab.ID, pending) })
	r.events.Publish(Event{Move: &MovedTab{WindowID: targetWindowID, MoveID: pending.id, Tab: tab}})
	return result
}

// Accept completes the move or the detach moveID of a tab into a window. A
// known tab changes owner to that window and keeps its group. A window with no
// watch keeps the tab until its grace period ends without a watch. An Accept
// repeated within the grace period after the move completed returns
// MoveErrorNone again.
func (r *Registry) Accept(windowID, tabID, moveID string) MoveError {
	r.mu.Lock()
	defer r.mu.Unlock()
	if windowID == "" || moveID == "" {
		return MoveErrorTimeout
	}
	if pending, ok := r.pendingMoves[tabID]; ok && pending.id == moveID && pending.targetWindowID == windowID {
		r.takeTab(windowID, tabID)
		r.endMove(tabID, pending, MoveErrorNone)
		r.recordAcceptedMove(moveID, windowID)
		return MoveErrorNone
	}
	if pending, ok := r.pendingDetaches[tabID]; ok && pending.id == moveID && pending.windowID == windowID {
		r.endDetach(tabID, pending, MoveErrorNone)
		r.takeTab(windowID, tabID)
		r.recordAcceptedMove(moveID, windowID)
		return MoveErrorNone
	}
	if r.acceptedMoves[moveID] == windowID {
		return MoveErrorNone
	}
	return MoveErrorTimeout
}

func (r *Registry) takeTab(windowID, tabID string) {
	tab, isKnown := r.tabs[tabID]
	if !isKnown {
		return
	}
	r.keepAcceptingWindow(windowID)
	r.setOwner(windowID, tab)
	r.windows[windowID] = appendMissing(r.windows[windowID], tabID)
	r.publishSnapshot()
}

func (r *Registry) recordAcceptedMove(moveID, windowID string) {
	r.acceptedMoves[moveID] = windowID
	r.schedule(r.grace, func() { r.forgetAcceptedMove(moveID) })
}

// CancelMove ends the pending move that result belongs to with
// MoveErrorCancelled. A move that already ended is left as it is.
func (r *Registry) CancelMove(tabID string, result <-chan MoveError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pending, ok := r.pendingMoves[tabID]; ok && (<-chan MoveError)(pending.result) == result {
		r.endMove(tabID, pending, MoveErrorCancelled)
	}
}

func (r *Registry) createMoveID() string {
	r.nextMoveID++
	return "move-" + strconv.Itoa(r.nextMoveID)
}

func (r *Registry) forgetAcceptedMove(moveID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.acceptedMoves, moveID)
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

func (r *Registry) expireMove(tabID string, pending *move) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingMoves[tabID] == pending {
		r.endMove(tabID, pending, MoveErrorTimeout)
	}
}

func (r *Registry) endMovesInto(windowID string) {
	for tabID, pending := range r.pendingMoves {
		if pending.targetWindowID == windowID {
			r.endMove(tabID, pending, MoveErrorWindowClosed)
		}
	}
	for tabID, pending := range r.pendingDetaches {
		if pending.windowID == windowID {
			r.endDetach(tabID, pending, MoveErrorWindowClosed)
		}
	}
}

func (r *Registry) endMove(tabID string, pending *move, outcome MoveError) {
	pending.cancel()
	delete(r.pendingMoves, tabID)
	pending.result <- outcome
}

func (r *Registry) endTransfers(tabID string, outcome MoveError) {
	if pending, ok := r.pendingMoves[tabID]; ok {
		r.endMove(tabID, pending, outcome)
	}
	if pending, ok := r.pendingDetaches[tabID]; ok {
		r.endDetach(tabID, pending, outcome)
	}
}
