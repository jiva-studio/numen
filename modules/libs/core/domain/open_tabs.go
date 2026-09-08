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
	Document *OpenDocument
	// Recording is the recording the tab holds, and nothing in a tab holding
	// none.
	Recording *OpenRecording
	// Book is the book that reflows the tab holds, and nothing in a tab holding
	// none.
	Book *OpenBook
}

// An OpenDocument is the document a tab holds, as the person is reading it.
type OpenDocument struct {
	// Page is the page in front of them, counted from one.
	Page int
	// Pages is how many pages the document has.
	Pages int
}

// An OpenBook is the book that reflows a tab holds, as the person is reading
// it. Such a book has no pages of its own, so where the person is is an offset
// into its text.
type OpenBook struct {
	// Offset is where they are reading, in bytes of the book's text.
	Offset int
	// Page is the page the offset falls on, counted from one, and Pages how many
	// the book is read in.
	Page  int
	Pages int
}

// An OpenRecording is the recording a tab holds, as far as it has been written
// down. Both are milliseconds.
type OpenRecording struct {
	// Heard is how much of it has been written down.
	Heard int
	// Length is how long the recording is.
	Length int
}

// The kinds of tab this application has words for. A tab of any other kind is
// spoken about by the name its window gave it.
const (
	TabNote      = "note"
	TabDocument  = "document"
	TabBook      = "book"
	TabRecording = "recording"
	TabPlex      = "plex"
)
