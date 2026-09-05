package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Agent takes tasks. It works a vault on a person's behalf, through tools: the
// window asks, and something outside this application answers.
//
// What answers is a program of somebody else's making, reached over a protocol
// of its own. Everything above this interface sees one agent and never which.
type Agent interface {
	// Take gives the agent a task and hands back the run it has begun.
	Take(ctx context.Context, task Task) (Run, error)
	// Finish says a conversation is over. What the agent kept of it is let go
	// of, and whatever is still being worked in it is stopped and waited for.
	// Empty is no conversation, and there is nothing to finish.
	Finish(ctx context.Context, conversation string) error
}

// Task is what the person asked, where they were looking when they asked it,
// and which conversation they asked it in.
type Task struct {
	Question string
	// Focus is the note the window is showing, empty when it shows none.
	Focus string
	// Conversation is which thread of talk this question belongs to, named by
	// the caller. Questions carrying one name are answered in one conversation
	// with the agent, so the name has to be the caller's alone: unique among
	// the conversations it has open, and never given to a second one for as
	// long as this process runs. Empty is no conversation, and a question
	// asked under it is answered on its own.
	Conversation string
}

// Run is one task being worked. Task is the request; this is its execution.
type Run interface {
	// Steps arrive in the order the agent takes them, and the channel closes
	// when there are no more.
	Steps() <-chan Step
	// Stop ends the run and returns once nothing of it is still running.
	Stop() error
}

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

// Step is one thing the agent said, did, or stopped for.
type Step struct {
	Kind StepKind
	// Call is what the agent named this call. Every step of one call carries the
	// same name, and a step that is not a call carries none.
	Call string
	// Text is what the agent is saying, for a step that says something.
	Text string
	// Tool is what it is using, by the name it is served under.
	Tool string
	// About is what that call is about — a note, a query — empty when the
	// call says nothing worth showing.
	About string
	// Place is where in the vault the call is working, empty when what the
	// call is about is not a source the vault holds.
	Place domain.Place
	// Count is how much of the call has been written, in characters. A call
	// carrying the text of a note is written for minutes, and this is the only
	// thing that moves while it is.
	Count int
	// Detail is why the work stopped, empty when the agent was done.
	Detail string
}
