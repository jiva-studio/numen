package completionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = domain.ErrUnavailable
)

// Question is what a plugin sends to a model.
type Question struct {
	Messages []Message
	Purpose  string
}

// Message is one turn of the conversation a question carries.
type Message struct {
	Role string
	Text string
}

// Completer answers a question of a plugin with text, from the default agent
// of the settings.
type Completer interface {
	// Complete answers the question of the plugin with the given id. A question
	// no agent answers fails with completion.ErrNoRoute, and an unreachable agent with
	// ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question Question) (string, error)
}

// CompleterHost receives the completer that answers plugin questions.
type CompleterHost interface {
	// SetCompleter sets the completer that answers plugin questions.
	SetCompleter(completer Completer)
}
