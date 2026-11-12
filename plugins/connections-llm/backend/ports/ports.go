// Package ports declares what the use cases of the plugin require of the service a connection names,
// in the words of the use cases.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// Gateway sends the requests of a call to the service its connection names and maps what the service
// answers: a failure is a try-next or a stop error with a reason, and the error of the context of the
// call is returned as it is. limit is how long the service may take.
type Gateway interface {
	// Complete asks for one answer to the messages.
	Complete(ctx context.Context, call connection.Call, limit time.Duration, request connection.ProviderCompleteRequest) (connection.CompleteResult, error)
	// Chat asks for the next turn of a conversation and passes its events to send as they arrive.
	Chat(ctx context.Context, call connection.Call, limit time.Duration, request connection.ProviderChatRequest, send func(connection.ChatEvent) error) error
	// Embed asks for one vector for each text.
	Embed(ctx context.Context, call connection.Call, limit time.Duration, texts []string) ([][]float32, error)
	// ListModels lists the models of the service that fit the operation.
	ListModels(ctx context.Context, call connection.Call, limit time.Duration, operation string) ([]connection.Model, error)
}
