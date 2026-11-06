package service

// instructions is the text an agent receives about showing the person a place with these tools.
const instructions = "Call `window_tabs_get` before describing the active tab to the person, and call it " +
	"again when it matters: the person moves between tabs while you work. Open a place only where " +
	"the person asked to see it. A search returns several passages of one source: pass every passage " +
	"you cite to `source_focus`, and the window scrolls to the first."
