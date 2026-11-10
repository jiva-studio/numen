package completionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/completion"
)

// Completer answers a question of a plugin with text, from the default agent
// of the settings.
type Completer interface {
	// Complete answers the question of the plugin with the given id. A question
	// no agent answers fails with completion.ErrNoRoute, and an unreachable agent with
	// ErrUnavailable.
	Complete(ctx context.Context, pluginID string, question completion.Question) (string, error)
}

// CompleterHost receives the completer that answers plugin questions.
type CompleterHost interface {
	// SetCompleter sets the completer that answers plugin questions.
	SetCompleter(completer Completer)
}
