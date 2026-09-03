package domain

// Attention is what a person has open: every tab of their window, and which of
// them they are looking at.
type Attention struct {
	// Tabs are every tab open, in the order the person was last in them.
	Tabs []Tab
	// Front is the tab in front, by its id. Empty where the window has nothing
	// open.
	Front string
}

// Fronted is the tab the person is looking at. A window holding none answers
// with no tab at all.
func (a Attention) Fronted() (Tab, bool) {
	for _, one := range a.Tabs {
		if one.ID != "" && one.ID == a.Front {
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
	// At is where in what the tab holds the person stands, and Of how much
	// there is of it, both counted in whatever that thing is measured in: a
	// document in pages, counted from one, and a recording in milliseconds,
	// where At is how much of it has been written down.
	At int
	Of int
}

// The kinds of tab this application has words for. A tab of any other kind is
// spoken about by the name its window gave it.
const (
	TabNote      = "note"
	TabDocument  = "document"
	TabRecording = "recording"
	TabPlex      = "plex"
)
