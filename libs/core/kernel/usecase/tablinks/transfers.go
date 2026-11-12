package tablinks

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// Transfers moves a tab to another window, or to a window opened for it.
type Transfers struct {
	state shared
}

// Detach stores a tab for ReadDetached and keeps the tab in its group while it
// moves to a new window. It returns a channel that receives the result once:
// MoveErrorNone when the window that read the tab calls Accept. A tab that no
// window reports within the grace period is dropped and unlinked. It ends a
// pending move or detach of the same tab with MoveErrorCanceled.
func (t *Transfers) Detach(tab tablink.SavedTab) <-chan tablink.MoveError {
	result := make(chan tablink.MoveError, 1)
	s, release := t.state.hold()
	defer release()
	s.transit.endTransfers(tab.ID, tablink.MoveErrorCanceled)
	pending := s.transit.addDetach(tab, result)
	s.expiries.scheduleDetach(s, tab.ID, pending)
	return result
}

// ReadDetached returns the tab and the move id of a pending detach to the
// first window that reads it, and again to that window only.
func (t *Transfers) ReadDetached(windowID, tabID string) (tablink.SavedTab, string, bool) {
	s, release := t.state.hold()
	defer release()
	pending, ok := s.transit.getDetach(tabID)
	if !ok || windowID == "" || (pending.windowID != "" && pending.windowID != windowID) {
		return tablink.SavedTab{}, "", false
	}
	pending.windowID = windowID
	return pending.tab, pending.id, true
}

// CancelDetach cancels the pending detach of a tab and drops its stored state.
// A tab missing from the last report of its window is unlinked.
func (t *Transfers) CancelDetach(tabID string) {
	s, release := t.state.hold()
	defer release()
	pending, ok := s.transit.getDetach(tabID)
	if !ok {
		return
	}
	s.transit.endDetach(tabID, pending, tablink.MoveErrorCanceled)
	s.dropUnreportedTab(tabID)
}

// Move sends a tab to another window and returns a channel that receives the
// result once. The source window owns the tab until the target window calls
// Accept. It ends a pending move or detach of the same tab with
// MoveErrorCanceled. A known tab moves only from the window that owns it.
func (t *Transfers) Move(sourceWindowID, targetWindowID string, tab tablink.SavedTab) <-chan tablink.MoveError {
	result := make(chan tablink.MoveError, 1)
	s, release := t.state.hold()
	defer release()
	if targetWindowID == sourceWindowID || !s.roster.windows[targetWindowID].hasWatch() {
		result <- tablink.MoveErrorWindowClosed
		return result
	}
	if known, ok := s.roster.tabs[tab.ID]; ok && known.WindowID != sourceWindowID {
		result <- tablink.MoveErrorNotOwner
		return result
	}
	s.transit.endTransfers(tab.ID, tablink.MoveErrorCanceled)
	pending := s.transit.addMove(tab.ID, targetWindowID, result)
	s.expiries.scheduleMove(s, tab.ID, pending)
	s.events.Publish(Event{Move: &MovedTab{WindowID: targetWindowID, MoveID: pending.id, Tab: tab}})
	return result
}

// CancelMove ends the pending move that result belongs to with
// MoveErrorCanceled. A move that already ended is left as it is.
func (t *Transfers) CancelMove(tabID string, result <-chan tablink.MoveError) {
	s, release := t.state.hold()
	defer release()
	if pending, ok := s.transit.getMove(tabID); ok && (<-chan tablink.MoveError)(pending.result) == result {
		s.transit.endMove(tabID, pending, tablink.MoveErrorCanceled)
	}
}

// Accept completes the move or the detach moveID of a tab into a window. A
// known tab changes owner to that window and keeps its group. A window with no
// watch keeps the tab until its grace period ends without a watch. An Accept
// repeated within the grace period after the move completed returns
// MoveErrorNone again.
func (t *Transfers) Accept(windowID, tabID, moveID string) tablink.MoveError {
	s, release := t.state.hold()
	defer release()
	if windowID == "" || moveID == "" {
		return tablink.MoveErrorTimeout
	}
	if pending, ok := s.transit.getMove(tabID); ok && pending.id == moveID && pending.targetWindowID == windowID {
		s.owners.takeTab(windowID, tabID)
		s.transit.endMove(tabID, pending, tablink.MoveErrorNone)
		s.expiries.recordAccepted(s, moveID, windowID)
		return tablink.MoveErrorNone
	}
	if pending, ok := s.transit.getDetach(tabID); ok && pending.id == moveID && pending.windowID == windowID {
		s.transit.endDetach(tabID, pending, tablink.MoveErrorNone)
		s.owners.takeTab(windowID, tabID)
		s.expiries.recordAccepted(s, moveID, windowID)
		return tablink.MoveErrorNone
	}
	if s.transit.accepted[moveID] == windowID {
		return tablink.MoveErrorNone
	}
	return tablink.MoveErrorTimeout
}
