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
	// Calling names a tool the agent is using.
	Calling
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
	// Failed is why the work stopped, empty when the agent was done.
	Failed string
}
