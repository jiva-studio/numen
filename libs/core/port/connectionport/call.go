package connectionport

import (
	"context"
	"errors"
	"strings"
)

// ErrNoConnection is a plugin with no row that can serve the operation.
var ErrNoConnection = errors.New("no connection serves the operation")

// Failure is one row that did not answer: the name of its connection and the reason.
type Failure struct{ ConnectionName, Reason string }

// AllFailedError is every row of a plugin set aside or failed.
type AllFailedError struct{ Failures []Failure }

// Error names each connection and its reason, in the order of the rows.
func (e *AllFailedError) Error() string {
	parts := make([]string, len(e.Failures))
	for i, one := range e.Failures {
		parts[i] = one.ConnectionName + ": " + one.Reason
	}
	return strings.Join(parts, "; ")
}

// RejectedError is a connection that found the request itself wrong.
type RejectedError struct{ ConnectionName, Reason string }

// Error names the connection and the reason.
func (e *RejectedError) Error() string { return e.ConnectionName + ": " + e.Reason }

// EmbeddingProfile is what the first row that answers says about its embedding model. Model is the
// model of that row. MaxTokens is zero when the limit is not known.
type EmbeddingProfile struct {
	Model          string
	Source         string
	Pooling        string
	QueryPrefix    string
	DocumentPrefix string
	Dimensions     int
	MaxTokens      int
}

// Priority says whose action a call serves. The zero value is background, so a call that sets none
// waits behind the calls of a person's actions.
type Priority int

const (
	// PriorityBackground is work that serves no person's action, such as indexing.
	PriorityBackground Priority = iota
	// PriorityInteractive is a person's action, such as a search or a check of a line.
	PriorityInteractive
)

// ModelCalls carries out the operations plugins ask for. The connections come from the rows of the
// calling plugin, and a request names none. At most as many calls as the field parallelRequests of a
// connection run at once; the rest wait, the interactive ones first, and a wait that times out sends the
// call to the next row. GetEmbeddingProfile is background.
type ModelCalls interface {
	// Complete asks the rows in order. It returns ErrNoConnection, *AllFailedError or *RejectedError where
	// no row answers, and the error of the context when the call is canceled.
	Complete(ctx context.Context, callerPluginID string, request CompleteRequest, priority Priority) (CompleteResult, error)
	// Chat asks the rows in order for the next turn of a conversation and sends its events to send as they
	// arrive. A row that fails before its first event is set aside and the next row is asked; a row that
	// fails after it ends the call with *AllFailedError naming that connection, and nothing is repeated.
	// It returns the errors Complete does, and the error of send when send fails.
	Chat(ctx context.Context, callerPluginID string, request ChatRequest, priority Priority, send func(ChatEvent) error) error
	// Embed asks the rows in order and returns the vectors in the order of the texts. It returns the errors
	// Complete does.
	Embed(ctx context.Context, callerPluginID string, texts []string, priority Priority) ([][]float32, error)
	// GetEmbeddingProfile asks the rows in order for the profile of their embedding model, measuring the
	// width where the provider has no profile. It returns the errors Complete does.
	GetEmbeddingProfile(ctx context.Context, callerPluginID string) (EmbeddingProfile, error)
}

// ModelCallsHost receives the calls plugins ask for.
type ModelCallsHost interface {
	// SetModelCalls sets what carries out the operations of the plugins.
	SetModelCalls(calls ModelCalls)
	// GetModelCalls returns what carries out the operations of the plugins, or nil where none is set.
	GetModelCalls() ModelCalls
}
