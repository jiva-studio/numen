package domain

// OpenTabs is what a person has open: every tab of their window, and which of
// them they are looking at.
type OpenTabs struct {
	// Tabs are every tab open, in the order the person was last in them.
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
	// Title is what the tab is called, as the person reads it.
	Title string
	// State is the JSON the plugin owning the kind saved, empty where it saved none.
	State string
}
