// Package tablink defines the tabs that windows of one vault link, move and detach.
package tablink

// Tab is a linkable tab open in a window.
type Tab struct {
	ID, Kind, Title string
	WindowID        string
}

// SavedTab is a tab with the state its kind saved, as the wire Tab. State is
// the JSON from the getState of its kind, empty when the kind saves none.
type SavedTab struct {
	ID, Kind, Path, Title, State string
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
	// the move timeout, or an Accept for a move no longer pending.
	MoveErrorTimeout
	// MoveErrorNotOwner is a source window that does not own the tab.
	MoveErrorNotOwner
	// MoveErrorCanceled is a move that a newer move or detach of the tab, a
	// cancel, or another window that took the tab ended.
	MoveErrorCanceled
)
