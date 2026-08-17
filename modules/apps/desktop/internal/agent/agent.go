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
}

// Task is what the person asked, and where they were looking when they asked
// it.
type Task struct {
	Asked string
	// Focus is the note the window is showing, empty when it shows none.
	Focus string
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
type Kind int

const (
	// Saying carries a piece of what the agent is telling the person.
	Saying Kind = iota
	// Calling names a tool the agent is using. It arrives more than once for one
	// call: the tool is named as soon as it is reached for, and again as its
	// arguments are written, because writing them is most of the wait.
	Calling
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
	// Text is what the agent is saying, for a step that says something.
	Text string
	// Tool is what it is using, by the name it is served under.
	Tool string
	// About is what that call is about — a note, a query — empty when the
	// call says nothing worth showing.
	About string
	// Written is how much of the call has been written, in characters. A call
	// carrying the text of a note is written for minutes, and this is the only
	// thing that moves while it is.
	Written int
	// Failed is why the work stopped, empty when the agent was done.
	Failed string
}
