// Package agent is what an agent is, in the words everything above it uses.
//
// An agent works a vault on a person's behalf, through tools. The window asks;
// something outside this application answers.
package agent

import "context"

// Agent takes tasks.
//
// What answers is a program of somebody else's making, reached over a protocol
// of its own. Everything above this interface sees one agent and never which.
type Agent interface {
	// Take gives the agent a task and hands back the work it has begun.
	Take(ctx context.Context, task Task) (Work, error)
	// Finish says a conversation is over. What the agent kept of it is let go
	// of, and whatever is still being worked in it is stopped and waited for.
	// Empty is no conversation, and there is nothing to finish.
	Finish(ctx context.Context, conversation string) error
}

// Task is what the person asked, where they were looking when they asked it,
// and which conversation they asked it in.
type Task struct {
	Asked string
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

// Work is one task being worked.
type Work interface {
	// Steps arrive in the order the agent takes them, and the channel closes
	// when there are no more.
	Steps() <-chan Step
	// Stop ends the work and returns once nothing of it is still running.
	Stop() error
}

// Kind is what a step is.
//
// A step naming a call says what that call does to the vault. A call whose kind
// is not known is Calling.
type Kind int

const (
	// Calling names a tool the agent is using. It arrives more than once for one
	// call: the tool is named as soon as it is reached for, and again as its
	// arguments are written, because writing them is most of the wait.
	Calling Kind = iota
	// Read names a call that reads the vault and leaves it as it was.
	Read
	// Edit names a call that writes a note.
	Edit
	// Remove names a call that takes a note out of the vault.
	Remove
	// Move names a call that files a note elsewhere.
	Move
	// Search names a call that looks for notes.
	Search
	// Saying carries a piece of what the agent is telling the person.
	Saying
	// Answered says the tool is finished. What follows is not this application's
	// and is not quick.
	Answered
	// Thinking says a request to the model has begun. It is the moment a wait
	// starts, and the only step that says nothing is being done here.
	Thinking
	// Stopped is the last step of any work.
	Stopped
)

// Step is one thing the agent said, did, or stopped for.
type Step struct {
	Kind Kind
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
	// call is about is not a note.
	Place Place
	// Written is how much of the call has been written, in characters. A call
	// carrying the text of a note is written for minutes, and this is the only
	// thing that moves while it is.
	Written int
	// Failed is why the work stopped, empty when the agent was done.
	Failed string
}

// Place is where a call is working: a note, by its path, and the place in it
// the call names.
type Place struct {
	Path string
	// Line is counted from one. Zero is a call that named the note and no place
	// in it.
	Line int
}
