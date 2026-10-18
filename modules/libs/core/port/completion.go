package port

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = errors.New("unavailable")

	// ErrNoRoute means no agent is routed for the purpose of a question. It is an
	// unavailability: the question has nowhere to go.
	ErrNoRoute = fmt.Errorf("%w: no route for the purpose", ErrUnavailable)
)

// Question is what a plugin asks of a model.
type Question struct {
	// Messages are the conversation so far, oldest first.
	Messages []Message
	// Purpose is what the answer is for. The core routes by it.
	Purpose string
	// Hints are what the answer should favour.
	Hints Hints
}

// Message is one turn of the conversation a question carries.
type Message struct {
	// Role is who wrote the text: "system", "user" or "assistant".
	Role string
	Text string
}

// Hints are priorities in the range 0 to 1. They are requests, and an agent may
// ignore them.
type Hints struct {
	QualityPriority float64
	SpeedPriority   float64
	CostPriority    float64
	// MaxTokens is the most tokens the answer may hold. Zero leaves the limit to
	// the agent.
	MaxTokens int
	// Temperature is the sampling temperature. Nil leaves it to the agent.
	Temperature *float64
}

// Completer answers a question with text, from whichever model the core routes
// its purpose to.
type Completer interface {
	// Complete answers the question. A question whose purpose has no route fails
	// with ErrNoRoute, and an agent that cannot be reached with ErrUnavailable.
	Complete(ctx context.Context, question Question) (string, error)
}

// CompletionAgents opens the agent a route names.
type CompletionAgents interface {
	// OpenCompletionAgent returns the agent named, or, when the name is empty,
	// the provider with the lowest id. An agent that is not there or cannot be
	// reached fails with ErrUnavailable.
	OpenCompletionAgent(ctx context.Context, agent string) (CompletionAgent, error)
}

// CompletionAgent is an agent answering one question at a time with a model of
// its own.
type CompletionAgent interface {
	// Complete answers the question with the model named. Empty leaves the
	// model to the agent.
	Complete(ctx context.Context, model string, question Question) (string, error)
	// Close lets the agent go. It may be called more than once.
	Close() error
}
