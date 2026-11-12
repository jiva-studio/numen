// Package model asks the connections of the plugin through the host, and holds the mock proofreader.
package model

import (
	"context"
	"errors"
	"os"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
)

// EnvMockML names the variable that has the plugin answer with the mock proofreader where its settings
// hold no connection; with a connection the plugin asks it.
const EnvMockML = "NUMEN_TEST_MOCK_ML"

const mockEnabled = "true"

// ErrNoConnection means the settings of the plugin hold no connection that can complete.
var ErrNoConnection = errors.New("proofreading is off: add a connection on the Connections page, then choose it in the settings of LLM Proofreading")

// Completer asks the connections of the plugin to complete a request.
type Completer func(ctx context.Context, request *llm.CompleteRequest, priority connection.Priority) (*llm.CompleteResult, error)

// NewCompleter returns the Completer that asks the rows of the plugin through caller.
func NewCompleter(caller connection.Caller) Completer {
	return func(ctx context.Context, request *llm.CompleteRequest, priority connection.Priority) (*llm.CompleteResult, error) {
		return llm.Complete(ctx, caller, request, priority)
	}
}

// IsMockEnabled reports whether the value of EnvMockML selects the mock model.
func IsMockEnabled(value string) bool { return value == mockEnabled }

// IsMockSelected reports whether the environment of the process selects the mock model.
func IsMockSelected() bool { return IsMockEnabled(os.Getenv(EnvMockML)) }

// NewAsk returns the function that asks the model one question. A person waits for the answer, so
// the request is interactive.
func NewAsk(complete Completer) func(ctx context.Context, instruction, prompt string) (string, error) {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		result, err := complete(ctx, &llm.CompleteRequest{
			Messages: []*llm.Message{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
		}, connection.PriorityInteractive)
		if errors.Is(err, connection.ErrNoConnection) {
			return "", ErrNoConnection
		}
		if err != nil {
			return "", err
		}
		return result.GetText(), nil
	}
}

// WithMock returns a Completer that answers with the mock proofreader where the settings hold no
// connection, and asks complete otherwise.
func WithMock(complete Completer) Completer {
	return func(ctx context.Context, request *llm.CompleteRequest, priority connection.Priority) (*llm.CompleteResult, error) {
		result, err := complete(ctx, request, priority)
		if errors.Is(err, connection.ErrNoConnection) {
			return CompleteMock(ctx, request)
		}
		return result, err
	}
}
