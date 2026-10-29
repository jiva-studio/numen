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
}
