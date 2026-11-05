package service

// instructions is the text an agent receives about showing the person a place with these tools.
const instructions = "Call `window_tabs_get` before saying what is in front of the person, and call it " +
	"again when it matters: the person moves between tabs while you work. Show a place only where " +
	"the person asked to be shown it. A search returns several places of one source: show every " +
	"place you speak about by passing them all to `source_focus`, and the person is taken to the first."
