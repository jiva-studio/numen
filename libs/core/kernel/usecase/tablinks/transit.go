package tablinks

import (
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

type move struct {
	id             string
	targetWindowID string
	result         chan tablink.MoveError
	cancel         func() bool
}

type detach struct {
	id       string
	tab      tablink.SavedTab
	windowID string
	result   chan tablink.MoveError
	cancel   func() bool
}

// transfer is the move or the detach that waits for a window to accept a tab. A tab has at most one.
type transfer struct {
	move   *move
	detach *detach
}

// transit is the moves and detaches that wait for a window to accept the tab, by the id of the tab, and
// the windows that accepted a move, by the id of the move.
type transit struct {
	transfers  map[string]*transfer
	nextMoveID int
	accepted   map[string]string
}

func newTransit() *transit {
	return &transit{transfers: map[string]*transfer{}, accepted: map[string]string{}}
}

func (t *transit) createMoveID() string {
	t.nextMoveID++
	return "move-" + strconv.Itoa(t.nextMoveID)
}

func (t *transit) getMove(tabID string) (*move, bool) {
	if pending := t.transfers[tabID]; pending != nil && pending.move != nil {
		return pending.move, true
	}
	return nil, false
}

func (t *transit) getDetach(tabID string) (*detach, bool) {
	if pending := t.transfers[tabID]; pending != nil && pending.detach != nil {
		return pending.detach, true
	}
	return nil, false
}

func (t *transit) ensure(tabID string) *transfer {
	pending := t.transfers[tabID]
	if pending == nil {
		pending = &transfer{}
		t.transfers[tabID] = pending
	}
	return pending
}

func (t *transit) addMove(tabID, targetWindowID string, result chan tablink.MoveError) *move {
	pending := &move{id: t.createMoveID(), targetWindowID: targetWindowID, result: result}
	t.ensure(tabID).move = pending
	return pending
}

func (t *transit) addDetach(tab tablink.SavedTab, result chan tablink.MoveError) *detach {
	pending := &detach{id: t.createMoveID(), tab: tab, result: result}
	t.ensure(tab.ID).detach = pending
	return pending
}

func (t *transit) isTransferringInto(windowID, tabID string) bool {
	pending := t.transfers[tabID]
	return pending != nil &&
		((pending.move != nil && pending.move.targetWindowID == windowID) ||
			(pending.detach != nil && pending.detach.windowID == windowID))
}

func (t *transit) endMove(tabID string, pending *move, outcome tablink.MoveError) {
	pending.cancel()
	if entry := t.transfers[tabID]; entry != nil {
		entry.move = nil
		if entry.detach == nil {
			delete(t.transfers, tabID)
		}
	}
	pending.result <- outcome
}

func (t *transit) endDetach(tabID string, pending *detach, outcome tablink.MoveError) {
	pending.cancel()
	if entry := t.transfers[tabID]; entry != nil {
		entry.detach = nil
		if entry.move == nil {
			delete(t.transfers, tabID)
		}
	}
	pending.result <- outcome
}

func (t *transit) endTransfers(tabID string, outcome tablink.MoveError) {
	if pending, ok := t.getMove(tabID); ok {
		t.endMove(tabID, pending, outcome)
	}
	if pending, ok := t.getDetach(tabID); ok {
		t.endDetach(tabID, pending, outcome)
	}
}

func (t *transit) endMovesInto(windowID string) {
	for tabID, entry := range t.transfers {
		if entry.move != nil && entry.move.targetWindowID == windowID {
			t.endMove(tabID, entry.move, tablink.MoveErrorWindowClosed)
		}
		if entry.detach != nil && entry.detach.windowID == windowID {
			t.endDetach(tabID, entry.detach, tablink.MoveErrorWindowClosed)
		}
	}
}
