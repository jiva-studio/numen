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

func (t *transit) createMoveID() string {
	t.nextMoveID++
	return "move-" + strconv.Itoa(t.nextMoveID)
}

func (t *transit) isMovingInto(windowID, tabID string) bool {
	pending, ok := t.moves[tabID]
	return ok && pending.targetWindowID == windowID
}

func (t *transit) isDetachingInto(windowID, tabID string) bool {
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
