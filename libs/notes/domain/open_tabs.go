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

// GetFrontTab is the tab the person is looking at. A window holding none
// answers with no tab at all.
func (o OpenTabs) GetFrontTab() (Tab, bool) {
	for _, one := range o.Tabs {
		if one.ID != "" && one.ID == o.FrontID {
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
