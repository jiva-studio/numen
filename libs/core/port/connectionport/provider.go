package connectionport

import (
	"context"
	"time"
)

// FailureReason is why a call to a provider failed. The provider names it from an enum; the core sees
// no HTTP code and no text from a service. The zero value is FailureReasonUnknown.
type FailureReason int

// The reasons of a failure.
const (
	FailureReasonUnknown FailureReason = iota
	FailureReasonKeyRejected
	FailureReasonPaymentRequired
	FailureReasonRateLimited
	FailureReasonUnreachable
	FailureReasonTimeout
	FailureReasonBadRequest
)

// String returns the words the person reads for the reason.
func (r FailureReason) String() string {
	switch r {
	case FailureReasonKeyRejected:
		return "key rejected"
	case FailureReasonPaymentRequired:
		return "payment required"
	case FailureReasonRateLimited:
		return "rate limited"
	case FailureReasonUnreachable:
		return "unreachable"
	case FailureReasonTimeout:
		return "timeout"
	case FailureReasonBadRequest:
		return "bad request"
	case FailureReasonUnknown:
		return "unknown"
	}
	return "unknown"
}

// TryNextError is a connection that cannot serve now. RetryAfter is zero when the service gave no wait.
type TryNextError struct {
	Reason     FailureReason
	RetryAfter time.Duration
}

// Error returns the reason.
func (e *TryNextError) Error() string { return e.Reason.String() }

// StopError is a request that is wrong in itself, so no other connection is tried.
type StopError struct{ Reason FailureReason }

// Error returns the reason.
func (e *StopError) Error() string { return e.Reason.String() }

// ProviderCall is the connection one call to a provider is made for: a snapshot read at the start of
// the call. Type is the id the manifest of the plugin gives the type, without the plugin id. Fields are the
// fields that are not secret, with the fixed values of the type, and Secrets are the values of the secret
// fields by key. The provider holds it for the length of the call.
type ProviderCall struct {
	Type      string
	Fields    map[string]string
	Secrets   map[string]string
	Model     string
	Reasoning string
}

// Message is one message of a completion. Role is system, user or assistant.
type Message struct{ Role, Text string }

// CompleteRequest is what a provider completes.
type CompleteRequest struct {
	Messages    []Message
	Temperature *float64
	MaxTokens   int
}

// CompleteResult is the text of an answer.
type CompleteResult struct{ Text string }

// ToolCall is a call of a tool the model asked for. Arguments is one JSON object.
type ToolCall struct{ ID, Name, Arguments string }

// ToolDefinition is a tool offered to the model. Parameters is the JSON schema of the arguments.
type ToolDefinition struct{ Name, Description, Parameters string }

// ChatMessage is one message of a conversation. Role is system, user, assistant or tool. An assistant
// message carries the calls it asked for, and a tool message carries the id of the call it answers.
type ChatMessage struct {
	Role       string
	Text       string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ChatRequest is one turn a provider carries out. The history is in Messages.
type ChatRequest struct {
	Messages    []ChatMessage
	Tools       []ToolDefinition
	Temperature *float64
	MaxTokens   int
}

// ChatFinishReason is why a turn ended. The zero value is ChatFinishStop.
type ChatFinishReason int

// The reasons a turn ends.
const (
	ChatFinishStop ChatFinishReason = iota
	ChatFinishToolCalls
	ChatFinishLength
)

// ChatEvent is one event of a turn: exactly one field is set. A text delta is never empty.
type ChatEvent struct {
	TextDelta string
	ToolCall  *ToolCall
	Finish    *ChatFinishReason
}

// Model is a model a service lists. Operations is empty when the service gives no label.
type Model struct {
	ID         string
	Title      string
	Operations []string
}

// ReasoningLevel is a level of reasoning a model takes.
type ReasoningLevel struct{ Value, Title string }

// ModelProfile is what a provider knows about an embedding model. IsKnown is false for a model it has
// no profile for. MaxTokens is zero when the limit is not known.
type ModelProfile struct {
	IsKnown        bool
	Dimensions     int
	MaxTokens      int
	Pooling        string
	QueryPrefix    string
	DocumentPrefix string
	Source         string
}

// The roles of a provider. A call fails with *TryNextError or *StopError, with the error of the context
// when it was canceled, or with an error of its own when the provider could not be reached.
type (
	// CompletionProvider carries out the operation complete.
	CompletionProvider interface {
		Complete(ctx context.Context, call ProviderCall, request CompleteRequest) (CompleteResult, error)
	}
	// EmbeddingProvider carries out the operation embed and returns the vectors in the order of the texts.
	EmbeddingProvider interface {
		Embed(ctx context.Context, call ProviderCall, texts []string) ([][]float32, error)
	}
	// ChatProvider carries out the operation chat. It sends the events of one turn in order and ends with
	// one finish. It returns the error of the first send that fails.
	ChatProvider interface {
		Chat(ctx context.Context, call ProviderCall, request ChatRequest, send func(ChatEvent) error) error
	}
	// ModelCatalog lists the models of a connection and the reasoning levels of the model of the call.
	ModelCatalog interface {
		ListModels(ctx context.Context, call ProviderCall, operation string) ([]Model, error)
		ListReasoningLevels(ctx context.Context, call ProviderCall) ([]ReasoningLevel, error)
	}
	// ModelProfiles reads the profile of an embedding model and measures the width of its vectors.
	ModelProfiles interface {
		GetModelProfile(ctx context.Context, call ProviderCall) (ModelProfile, error)
		ProbeEmbeddingWidth(ctx context.Context, call ProviderCall) (int, error)
	}
)

// Provider is what a plugin that provides connection types carries out.
type Provider interface {
	CompletionProvider
	ChatProvider
	EmbeddingProvider
	ModelCatalog
	ModelProfiles
}

// ProviderSource opens the provider of a plugin.
type ProviderSource interface {
	// OpenProvider starts the plugin if it is not running and returns its provider with the function that
	// releases it. A plugin that cannot be started is an error.
	OpenProvider(ctx context.Context, pluginID string) (Provider, func(), error)
}

// Row is one entry of the settings of a plugin that names a connection, with the model and the
// reasoning level chosen for it.
type Row struct{ ConnectionID, Model, Reasoning string }

// Rows reads the rows of a plugin in the order of use.
type Rows interface {
	// ListRows returns the rows of the plugin for an operation, in the order the person set.
	ListRows(pluginID, operation string) ([]Row, error)
}

// Notices tells the person what the core did on its own.
type Notices interface {
	// ReportSetAside says that a connection was set aside for a reason and a row after it answered.
	ReportSetAside(connectionID, connectionName string, reason FailureReason)
}
