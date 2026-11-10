package window

// OpenTabs is the set of tabs open in the user's window and the tab that is
// active.
type OpenTabs struct {
	Tabs     []Tab
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

// A Tab is one tab of a window: what kind it is, and its contents.
//
// The kind is the name a plugin registers the tab kind under. State is the
// JSON that plugin saved for the tab; nothing here reads it.
type Tab struct {
	ID    string
	Kind  string
	Path  string
	Title string
	State string
	Place *TabPlace
}

// TabPlace is the place a reader tab shows, as its reader reports it. A zero field is a place
// the reader does not have.
type TabPlace struct {
	Page            int
	PageCount       int
	PositionSeconds *float64
	CFI             string
	Chapter         string
}
