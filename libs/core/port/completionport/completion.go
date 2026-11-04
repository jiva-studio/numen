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

	// ErrInvalidQuestion means the question holds nothing an agent can answer.
	ErrInvalidQuestion = errors.New("invalid question")
)

// Question is what a plugin sends to a model.
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

// Completer answers a question of a plugin with text, from the default agent
// of the settings.
type Completer interface {
	// Complete answers the question of the plugin with the given id. A question
	// no agent answers fails with ErrNoRoute, and an unreachable agent with
	// ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question Question) (string, error)
}

// CompleterHost receives the completer that answers plugin questions.
type CompleterHost interface {
	// SetCompleter sets the completer that answers plugin questions.
	SetCompleter(completer Completer)
}
