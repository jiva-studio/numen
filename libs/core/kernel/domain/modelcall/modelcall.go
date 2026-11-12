// Package modelcall holds the vocabulary of the calls on models that plugins ask for: the requests
// and answers of a completion, a chat turn and an embedding, and the models a service lists.
package modelcall

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
