package completionport

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = domain.ErrUnavailable

	// ErrNoRoute means no agent answers the question: the program the settings
	// name is not installed or no plugin lists it, or no program is installed at
	// all. It is an unavailability: the question has nowhere to go.
	ErrNoRoute = fmt.Errorf("%w: no agent answers the question", ErrUnavailable)

	// ErrNoConsent means the person has not let the plugin ask for completions,
	ErrNoConsent = errors.New("AI access not allowed — allow it in Settings → AI")

	// ErrInvalidQuestion means the question holds nothing an agent can answer.
	ErrInvalidQuestion = errors.New("invalid question")
)

// Question is what a plugin asks of a model.
type Question struct {
	// Messages are the conversation so far, oldest first.
	Messages []Message
	// Purpose is what the answer is for. It routes nothing.
	Purpose string
}

// Message is one turn of the conversation a question carries.
type Message struct {
	// Role is who wrote the text: "system", "user" or "assistant".
	Role string
	Text string
}

// CompletionChoice is the agent program and the model a plugin's completions go
// to, as the person set them in the plugin's setting of type SettingTypeAgent.
type CompletionChoice struct {
	// Agent is the name of the agent program. Empty is the default of the
	// settings.
	Agent string
	// Model is the model the program answers with. Empty is the one the settings
	// name for the program, else the program's own.
	Model string
}

// ReadCompletionChoice reads the value of a setting of type SettingTypeAgent. A
// value of any other shape is the empty choice.
func ReadCompletionChoice(value any) CompletionChoice {
	fields, ok := value.(map[string]any)
	if !ok {
		return CompletionChoice{}
	}
	return CompletionChoice{Agent: getText(fields, "agent"), Model: getText(fields, "model")}
}

// getText is the string at the key, and empty where the key holds none.
func getText(fields map[string]any, key string) string {
	text, isText := fields[key].(string)
	if !isText {
		return ""
	}
	return text
}

// Completer answers a question of a plugin with text, from the agent the
// plugin's setting names, else the default of the settings.
type Completer interface {
	// Complete answers the question the plugin with the id given asks. A
	// question no agent answers fails with ErrNoRoute, and an agent that cannot
	// be reached with ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question Question) (string, error)
}

// CompleterHost receives the completer that answers plugin questions.
type CompleterHost interface {
	// SetCompleter sets the completer that answers plugin questions.
	SetCompleter(completer Completer)
}

// CompletionChoices reads the choice a plugin's setting holds.
type CompletionChoices interface {
	// GetCompletionChoice returns the agent and the model the plugin's setting of
	// type SettingTypeAgent holds, and the empty choice where it holds none.
	GetCompletionChoice(pluginID string) CompletionChoice
}
