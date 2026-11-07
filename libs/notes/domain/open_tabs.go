package domain

// OpenTabs is the set of tabs open in the user's window and the tab that is
// active.
type OpenTabs struct {
	// Tabs are every open tab, in the order they were last active.
	Tabs []Tab
	// ActiveID is the id of the active tab. Empty where the window has
	// nothing open.
	ActiveID string
}

// GetActiveTab returns the active tab, and false where the window has none.
func (o OpenTabs) GetActiveTab() (Tab, bool) {
	for _, one := range o.Tabs {
		if one.ID != "" && one.ID == o.ActiveID {
			return one, true
		}
	}
	return Tab{}, false
}

// A Tab is one tab of a window: what kind it is, and what it holds.
//
// The kind is the name a plugin registers the tab kind under. State is the
// JSON that plugin saved for the tab; nothing here reads it.
type Tab struct {
	ID   string
	Kind string
	// Path is the file the tab holds, empty for a tab holding no file.
	Path string
	// Title is the display title of the tab.
	Title string
	// State is the JSON the plugin owning the kind saved, empty where it saved none.
	State string
	// Place is the place the reader of the tab reported, nil for a tab whose kind reports none.
	Place *TabPlace
}

// TabPlace is the place a reader tab shows, as its reader reports it. A zero field is a place
// the reader does not have.
type TabPlace struct {
	// Page is the page a PDF shows, from 1.
	Page int
	// PageCount is the number of pages of a PDF.
	PageCount int
	// PositionSeconds is the time of a recording, in seconds from its start; nil for a tab that
	// is not a recording.
	PositionSeconds *float64
	// CFI is the EPUB CFI of the place shown.
	CFI string
	// Chapter is the table-of-contents title of an EPUB at the place shown.
	Chapter string
}
