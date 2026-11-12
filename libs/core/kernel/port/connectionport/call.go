package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// ModelCalls carries out the operations plugins ask for. The connections come from the rows of the
// calling plugin, and a request names none. At most as many calls as the field parallelRequests of a
// connection run at once; the rest wait, the interactive ones first, and a wait that times out sends the
// call to the next row. GetEmbeddingProfile is background.
type ModelCalls interface {
	// Complete asks the rows in order. It returns connection.ErrNoConnection, *connection.AllFailedError or
	// *connection.RejectedError where no row answers, and the error of the context when the call is canceled.
	Complete(ctx context.Context, callerPluginID string, request modelcall.CompleteRequest, priority modelcall.Priority) (modelcall.CompleteResult, error)
	// Chat asks the rows in order for the next turn of a conversation and sends its events to send as they
	// arrive. A row that fails before its first event is set aside and the next row is asked; a row that
	// fails after it ends the call with *connection.AllFailedError naming that connection, and nothing is
	// repeated. It returns the errors Complete does, and the error of send when send fails.
	Chat(ctx context.Context, callerPluginID string, request modelcall.ChatRequest, priority modelcall.Priority, send func(modelcall.ChatEvent) error) error
	// Embed asks the rows in order and returns the vectors in the order of the texts. It returns the errors
	// Complete does.
	Embed(ctx context.Context, callerPluginID string, texts []string, priority modelcall.Priority) ([][]float32, error)
	// GetEmbeddingProfile asks the rows in order for the profile of their embedding model, measuring the
	// width where the provider has no profile. It returns the errors Complete does.
	GetEmbeddingProfile(ctx context.Context, callerPluginID string) (modelcall.EmbeddingProfile, error)
}
