package service

// The instructions of the window tools, each one sentence. A sentence of a tool offered on the
// reading server names no tool offered on the full server alone.
const (
	tabListSentence  = "A task that says \"this note\" means the note in the active tab, which window_tab_list returns; " +
		"call it before you describe the active tab, and again before each later description, because the user switches tabs while you work."
	showSentence     = "note_focus, source_focus and window_tab_activate change what the window shows, so call them only when the user asked to see that place."
	passagesSentence = "A search returns several passages of one source; pass every passage you cite to source_focus, and the window scrolls to the first."
	tabIDSentence    = "window_tab_activate takes the id of a tab that window_tab_list listed."
)
