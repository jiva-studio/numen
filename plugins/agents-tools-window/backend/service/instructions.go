package service

// instructions is the text an agent receives about opening a place in the user's window with these tools.
const instructions = "Call `window_tabs_get` before describing the active tab to the user, and call it " +
	"again before each later description: the user switches tabs while you work. Open a place only when " +
	"the user asked to see it. A search returns several passages of one source: pass every passage " +
	"you cite to `source_focus`, and the window scrolls to the first."
