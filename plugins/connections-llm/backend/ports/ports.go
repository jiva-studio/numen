// Package ports declares what the use cases of the plugin require of the service a connection names,
// in the words of the use cases.
package ports

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// Gateway sends the requests of a call to the service its connection names and maps what the service
// answers: a failure is a try-next or a stop error with a reason, and the error of the context of the
// call is returned as it is. limit is how long the service may take. The model and the reasoning level
// are the options of the row of the call.
type Gateway interface {
	// Complete asks for one answer to the messages.
	Complete(ctx context.Context, call connection.Call, limit time.Duration, request *llm.CompleteRequest) (*llm.CompleteResult, error)
	// Chat asks for the next turn of a conversation, passes its frames to send as they arrive and returns
	// why the turn ended.
	Chat(ctx context.Context, call connection.Call, limit time.Duration, request *llm.ChatRequest, send func(*llm.ChatFrame) error) (*llm.ChatFinish, error)
	// Embed asks for one vector for each text.
	Embed(ctx context.Context, call connection.Call, limit time.Duration, texts []string) ([][]float32, error)
	// Rerank asks for one score for each document, in the order of the documents.
	Rerank(ctx context.Context, call connection.Call, limit time.Duration, request *llm.RerankRequest) ([]float32, error)
	// ListModels lists the models of the service that fit the operations of a setting of rows.
	ListModels(ctx context.Context, call connection.Call, limit time.Duration, operations []string) ([]connection.Option, error)
}
