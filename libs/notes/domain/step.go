package domain

// StepKind is what a step is. A call whose kind is not known is StepToolCall.
type StepKind int

const (
	// StepToolCall names a tool the agent is using, once as it is reached for and again as its arguments are written.
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
	// StepSay carries a piece of what the agent is telling the person.
	StepSay
	// StepToolDone says the tool is finished.
	StepToolDone
	// StepModelStart says a request to the model has begun.
	StepModelStart
	// StepDraft carries a change to a note while the call writing it is still arriving: the note, the
	// text the change replaces and the text that takes its place so far. It is drawn and never shown as a
	// step.
	StepDraft
	// StepStop is the last step of any work.
	StepStop
)

// ReasoningLevel is one depth of reasoning a model can be asked to work at.
type ReasoningLevel struct {
	// Name is what a task asks for the level by.
	Name string
	// Title is what the level is called when shown to a person.
	Title string
}
