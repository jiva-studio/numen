package tablinks

import "github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"

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
	Tabs    []tablink.Tab
	Windows []Window
}

// PathSync is the new path of a linked tab.
type PathSync struct {
	GroupID, SourceTabID, Path, Title string
}

// MovedTab is a tab sent to the window WindowID, as the wire MovedTab. The
// window passes MoveID to Accept.
type MovedTab struct {
	WindowID string
	MoveID   string
	Tab      tablink.SavedTab
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
