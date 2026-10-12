package domain

// StepKind is what a step is.
//
// A step naming a call says what that call does to the vault. A call whose kind
// is not known is StepToolCall.
type StepKind int

const (
	// StepToolCall names a tool the agent is using. It arrives more than once for
	// one call: the tool is named as soon as it is reached for, and again as its
	// arguments are written, because writing them is most of the wait.
	StepToolCall StepKind = iota
	// StepRead names a call that reads the vault and leaves it as it was.
	StepRead
	// StepEdit names a call that writes a note.
	StepEdit
	// StepRemove names a call that takes a note out of the vault.
	StepRemove
	// StepMove names a call that files a note elsewhere.
	StepMove
	// StepSearch names a call that looks for notes.
	StepSearch
	// StepSaying carries a piece of what the agent is telling the person.
	StepSaying
	// StepAnswered says the tool is finished. What follows is not this
	// application's and is not quick.
	StepAnswered
	// StepThinking says a request to the model has begun. It is the moment a
	// wait starts, and the only step that says nothing is being done here.
	StepThinking
	// StepStopped is the last step of any work.
	StepStopped
)
