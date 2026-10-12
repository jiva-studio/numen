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
	// StartRun gives the agent a task and hands back the run it has begun.
	StartRun(ctx context.Context, task Task) (Run, error)
	// EndConversation says a conversation is over. What the agent kept of it is let go
	// of, and whatever is still being worked in it is stopped and waited for.
	// Empty is no conversation, and there is nothing to finish.
	EndConversation(ctx context.Context, conversation string) error
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
	// Agent is which agent answers for this task (e.g. "antigravity", "codex", "claude").
	// Empty falls back to the configured default agent.
	Agent string
	// Model is which model answers for this task. Empty uses default.
	Model string
}

// Run is one task being worked. Task is the request; this is its execution.
type Run interface {
	// GetSteps returns the steps in the order the agent takes them. The channel
	// closes when there are no more.
	GetSteps() <-chan Step
	// Stop ends the run and returns once nothing of it is still running.
	Stop() error
}

// Step is one thing the agent said, did, or stopped for.
type Step struct {
	Kind domain.StepKind
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
