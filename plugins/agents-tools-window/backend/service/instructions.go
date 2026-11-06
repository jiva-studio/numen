package service

// permissionWindowFocus is the permission a caller holds to read the tabs of the window and change
// what it shows.
const permissionWindowFocus = "window:focus"

// The instructions of the window tools, each one sentence. A sentence of a tool offered on the
// reading server names no tool offered on the full server alone.
const (
	tabsGetSentence  = "Call window_tabs_get before you describe the active tab, and again before each later description, because the user switches tabs while you work."
	showSentence     = "note_focus, source_focus and window_focus_tab change what the window shows, so call them only when the user asked to see that place."
	passagesSentence = "A search returns several passages of one source; pass every passage you cite to source_focus, and the window scrolls to the first."
	tabIDSentence    = "window_focus_tab takes the id of a tab that window_tabs_get listed."
)
