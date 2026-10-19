package port

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = errors.New("unavailable")

	// ErrNoRoute means no agent answers the question: the agent the settings name
	// is not installed, is disabled or answers no completions, or no plugin
	// answers completions at all. It is an unavailability: the question has
	// nowhere to go.
	ErrNoRoute = fmt.Errorf("%w: no agent answers the question", ErrUnavailable)

	// ErrNoConsent means the person has not let the plugin ask for completions,
	// or has refused it.
	ErrNoConsent = errors.New("AI access not allowed — allow it in Settings → AI")
)

// Question is what a plugin asks of a model.
type Question struct {
	// Messages are the conversation so far, oldest first.
	Messages []Message
	// Purpose is what the answer is for. It is handed to the agent and routes
	// nothing.
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

// CompletionChoice is the agent and the model a plugin's completions go to, as
// the person set them in the plugin's setting of type SettingTypeAgent.
type CompletionChoice struct {
	// Agent is the id of the agent plugin. Empty is the default of the settings.
	Agent string `json:"agent,omitempty"`
	// Model is the model the agent answers with. Empty is the agent's own.
	Model string `json:"model,omitempty"`
}

// ReadCompletionChoice reads the value of a setting of type SettingTypeAgent. A
// value of any other shape is the empty choice.
func ReadCompletionChoice(value any) CompletionChoice {
	held, ok := value.(map[string]any)
	if !ok {
		return CompletionChoice{}
	}
	agent, _ := held["agent"].(string)
	model, _ := held["model"].(string)
	return CompletionChoice{Agent: agent, Model: model}
}

// Completer answers a question of a plugin with text, from the agent the
// plugin's setting names, else the default of the settings.
type Completer interface {
	// Complete answers the question the plugin with the id given asks. A
	// question no agent answers fails with ErrNoRoute, and an agent that cannot
	// be reached with ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question Question) (string, error)
}

// CompletionChoices reads the choice a plugin's setting holds.
type CompletionChoices interface {
	// GetCompletionChoice returns the agent and the model the plugin's setting of
	// type SettingTypeAgent holds, and the empty choice where it holds none.
	GetCompletionChoice(pluginID string) CompletionChoice
}

// CompletionAgents opens the agent a route names.
type CompletionAgents interface {
	// OpenCompletionAgent returns the agent named, or, when the name is empty,
	// the provider with the lowest id. An agent that is not there, or answers no
	// completions, fails with ErrNoRoute, and one that cannot be reached with
	// ErrUnavailable.
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
