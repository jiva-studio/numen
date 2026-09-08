package domain

// OpenTabs is what a person has open: every tab of their window, and which of
// them they are looking at.
type OpenTabs struct {
	// Tabs are every tab open, in the order the person was last in them.
	Tabs []Tab
	// FrontID is the tab in front, by its id. Empty where the window has
	// nothing open.
	FrontID string
}

// Fronted is the tab the person is looking at. A window holding none answers
// with no tab at all.
func (o OpenTabs) Fronted() (Tab, bool) {
	for _, one := range o.Tabs {
		if one.ID != "" && one.ID == o.FrontID {
			return one, true
		}
	}
	return Tab{}, false
}

// A Tab is one tab of a window: what kind it is, and what it holds.
//
// The kind is the window's own word for it. A window is free to open a kind
// nothing here has heard of, and what such a tab holds is its kind and no more.
type Tab struct {
	ID   string
	Kind string
	// Path is the file the tab holds, empty for a tab holding no file. A plex
	// holds the note it stands on.
	Path string
	// Title is what the tab is called, as the person reads it.
	Title string
	// Document is the document the tab holds, and nothing in a tab holding
	// none.
	Document *DocumentProgress
	// Recording is the recording the tab holds, and nothing in a tab holding
	// none.
	Recording *RecordingProgress
}

// A DocumentProgress is how far through a document the person reading it is.
type DocumentProgress struct {
	// Page is the page in front of them, counted from one.
	Page int
	// PageCount is how many pages the document has.
	PageCount int
}

// A RecordingProgress is how far into a recording the words written down reach.
// Both are milliseconds.
type RecordingProgress struct {
	// TranscribedDuration is how far into the recording the words written down
	// reach. It is short of the duration while a run is still listening.
	TranscribedDuration int
	// Duration is how long the recording is.
	Duration int
}

// The kinds of tab this application has words for. A tab of any other kind is
// spoken about by the name its window gave it.
const (
	TabNote      = "note"
	TabDocument  = "document"
	TabRecording = "recording"
	TabPlex      = "plex"
)
