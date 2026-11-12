package connection

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// Call is the connection one call of a provider is made for. The provider holds it for the length of the
// call and keeps neither it nor a key.
//
//numen:api
type Call struct {
	// Type is the id of the connection type in the manifest of the plugin, without the plugin id.
	Type string
	// Fields are the fields that are not secret, with the fixed values of the type.
	Fields map[string]string
	// Secrets are the secret fields by key. A Secret prints without its value.
	Secrets map[string]Secret
	// Model is the name the person chose. The provider does not check it against its list of models.
	Model string
	// Reasoning is a value from ListReasoningLevels, or empty for the default of the provider.
	Reasoning string
}

// Message is one message of a completion. Role is system, user or assistant.
type Message struct{ Role, Text string }

// ProviderCompleteRequest is what a provider completes.
type ProviderCompleteRequest struct {
	Messages    []Message
	Temperature *float64
	MaxTokens   int
}

// CompleteResult is the text of an answer.
type CompleteResult struct{ Text string }

// Model is a model a service lists. Operations is empty when the service gives no label.
type Model struct {
	ID         string
	Title      string
	Operations []string
}

// ReasoningLevel is a level of reasoning a model takes.
type ReasoningLevel struct{ Value, Title string }

// ModelProfile is what a provider knows about an embedding model. IsKnown is false for a model it has
// no profile for, and the core then probes the width. MaxTokens is zero when the limit is not known.
type ModelProfile struct {
	IsKnown        bool
	Dimensions     int
	MaxTokens      int
	Pooling        string
	QueryPrefix    string
	DocumentPrefix string
	Source         string
}

// Completer carries out the operation complete.
type Completer interface {
	Complete(ctx context.Context, call Call, request ProviderCompleteRequest) (CompleteResult, error)
}

// Embedder carries out the operation embed. It returns raw vectors in the order of the texts.
type Embedder interface {
	Embed(ctx context.Context, call Call, texts []string) ([][]float32, error)
}

// ModelLister lists the models of a connection and the reasoning levels of the model of the call.
type ModelLister interface {
	ListModels(ctx context.Context, call Call, operation string) ([]Model, error)
	ListReasoningLevels(ctx context.Context, call Call) ([]ReasoningLevel, error)
}

// ProfileReader reads the profile of the embedding model of the call, and measures the width of its
// vectors where the profile is not known.
type ProfileReader interface {
	GetModelProfile(ctx context.Context, call Call) (ModelProfile, error)
	ProbeEmbeddingWidth(ctx context.Context, call Call) (int, error)
}

// Provider lists the roles one plugin fills. A nil role is not served.
//
// A role returns NewTryNextError or NewStopError for a call that failed, and ctx.Err() for a call that
// was canceled. Any other error is a stop of unknown reason.
type Provider struct {
	Completer Completer
	Embedder  Embedder
	Models    ModelLister
	Profiles  ProfileReader
	Chatter   Chatter
}

// NewHandler creates the Connect-RPC handler for the roles of a Provider.
func NewHandler(provider Provider, opts ...connect.HandlerOption) (string, http.Handler) {
	return numenv1connect.NewProviderServiceHandler(&adapter{provider: provider}, opts...)
}

// Mount registers the roles of a Provider on the plugin Server.
func Mount(s *server.Server, provider Provider) {
	s.Register(NewHandler(provider, s.HandlerOptions()...))
}
