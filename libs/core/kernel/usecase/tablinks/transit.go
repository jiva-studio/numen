package tablinks

import (
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// move is one pending move of a tab.
type move struct {
	id             string
	targetWindowID string
	result         chan tablink.MoveError
	cancel         func() bool
}

// detach is a pending detach: the tab with its saved state, the window that
// read it, the id that window passes to Accept, the channel of its result and
// the cancel function of its expiry.
type detach struct {
	id       string
	tab      tablink.SavedTab
	windowID string
	result   chan tablink.MoveError
	cancel   func() bool
}

// transit is the tabs on their way to another window.
type transit struct {
	detaches   map[string]*detach
	moves      map[string]*move
	nextMoveID int
	accepted   map[string]string
}

func newTransit() *transit {
	return &transit{
		detaches: map[string]*detach{},
		moves:    map[string]*move{},
		accepted: map[string]string{},
	}
}

// transitBook is the moves and detaches that wait for a window to accept the tab.
type transitBook interface {
	moveBook
	detachBook
	acceptBook
	transferFlow
}

// moveBook is the pending moves of tabs to other windows.
type moveBook interface {
	getMove(tabID string) (*move, bool)
	addMove(tabID, targetWindowID string, result chan tablink.MoveError) *move
	endMove(tabID string, pending *move, outcome tablink.MoveError)
}

// detachBook is the pending detaches of tabs to new windows.
type detachBook interface {
	getDetach(tabID string) (*detach, bool)
	addDetach(tab tablink.SavedTab, result chan tablink.MoveError) *detach
	endDetach(tabID string, pending *detach, outcome tablink.MoveError)
}

// acceptBook is the windows that accepted a move, by the id of the move.
type acceptBook interface {
	getAccepted(moveID string) (string, bool)
	putAccepted(moveID, windowID string)
	deleteAccepted(moveID string)
}

// transferFlow tells which transfers go to a window and ends them in bulk.
type transferFlow interface {
	isTransferringInto(windowID, tabID string) bool
	endTransfers(tabID string, outcome tablink.MoveError)
	endMovesInto(windowID string)
}

func (t *transit) getMove(tabID string) (*move, bool) {
	pending, ok := t.moves[tabID]
	return pending, ok
}

func (t *transit) addMove(tabID, targetWindowID string, result chan tablink.MoveError) *move {
	pending := &move{id: t.createMoveID(), targetWindowID: targetWindowID, result: result}
	t.moves[tabID] = pending
	return pending
}

func (t *transit) getDetach(tabID string) (*detach, bool) {
	pending, ok := t.detaches[tabID]
	return pending, ok
}

func (t *transit) addDetach(tab tablink.SavedTab, result chan tablink.MoveError) *detach {
	pending := &detach{id: t.createMoveID(), tab: tab, result: result}
	t.detaches[tab.ID] = pending
	return pending
}

func (t *transit) getAccepted(moveID string) (string, bool) {
	windowID, ok := t.accepted[moveID]
	return windowID, ok
}

func (t *transit) putAccepted(moveID, windowID string) { t.accepted[moveID] = windowID }

func (t *transit) deleteAccepted(moveID string) { delete(t.accepted, moveID) }

func (t *transit) createMoveID() string {
	t.nextMoveID++
	return "move-" + strconv.Itoa(t.nextMoveID)
}

func (t *transit) isTransferringInto(windowID, tabID string) bool {
	if pending, ok := t.moves[tabID]; ok && pending.targetWindowID == windowID {
		return true
	}
	pending, ok := t.detaches[tabID]
	return ok && pending.windowID == windowID
}

func (t *transit) endMove(tabID string, pending *move, outcome tablink.MoveError) {
	pending.cancel()
	delete(t.moves, tabID)
	pending.result <- outcome
}

func (t *transit) endDetach(tabID string, pending *detach, outcome tablink.MoveError) {
	pending.cancel()
	delete(t.detaches, tabID)
	pending.result <- outcome
}

func (t *transit) endTransfers(tabID string, outcome tablink.MoveError) {
	if pending, ok := t.moves[tabID]; ok {
		t.endMove(tabID, pending, outcome)
	}
	if pending, ok := t.detaches[tabID]; ok {
		t.endDetach(tabID, pending, outcome)
	}
}

func (t *transit) endMovesInto(windowID string) {
	for tabID, pending := range t.moves {
		if pending.targetWindowID == windowID {
			t.endMove(tabID, pending, tablink.MoveErrorWindowClosed)
		}
	}
	for tabID, pending := range t.detaches {
		if pending.windowID == windowID {
			t.endDetach(tabID, pending, tablink.MoveErrorWindowClosed)
		}
	}
}
