// Package tablink defines the tabs that windows of one vault link, move and detach: the groups they
// form, the windows that report them, and the events that tell the windows what changed.
package tablink

// Tab is a linkable tab open in a window.
type Tab struct {
	ID, Kind, Title string
	WindowID        string
}

// Group is two or more tabs that move to the same path together.
type Group struct {
	ID          string
	ColorIndex  int
	TabIDs      []string
	Path, Title string
}

// Window is the last report of a window about itself.
type Window struct {
	ID                      string
	IsMain                  bool
	ActiveTitle, ActiveKind string
	TabCount                int
}

// Snapshot holds every group, every known tab and every window that reported.
type Snapshot struct {
	Groups  []Group
	Tabs    []Tab
	Windows []Window
}

// PathSync is the new path of a linked tab.
type PathSync struct {
	GroupID, SourceTabID, Path, Title string
}

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

// Highlight turns the outline of the window WindowID on or off.
type Highlight struct {
	WindowID string
	IsOn     bool
}

// Event is one message to watching windows: a snapshot or a path sync for
// every window, or a moved tab or a highlight for one.
type Event struct {
	Snapshot  *Snapshot
	Sync      *PathSync
	Move      *MovedTab
	Highlight *Highlight
}

// IsFor reports whether a window receives the event.
func (e Event) IsFor(windowID string) bool {
	switch {
	case e.Move != nil:
		return e.Move.WindowID == windowID
	case e.Highlight != nil:
		return e.Highlight.WindowID == windowID
	default:
		return true
	}
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
	// MoveErrorCancelled is a move that a newer move or detach of the tab, a
	// cancel, or another window that took the tab ended.
	MoveErrorCancelled
)
