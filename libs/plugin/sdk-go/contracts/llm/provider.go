package llm

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

// Completer carries out llm.complete.
type Completer interface {
	Complete(ctx context.Context, call connection.Call, request *CompleteRequest) (*CompleteResult, error)
}

// Chatter carries out llm.chat. It sends the frames of one turn in order, text deltas and then the
// complete tool calls, and returns the finish. It returns the error of the first send that fails.
type Chatter interface {
	Chat(ctx context.Context, call connection.Call, request *ChatRequest, send func(*ChatFrame) error) (*ChatFinish, error)
}

// Embedder carries out llm.embed. It returns one vector for each text, in the order of the texts.
type Embedder interface {
	Embed(ctx context.Context, call connection.Call, request *EmbedRequest) (*EmbedResult, error)
}

// Reranker carries out llm.rerank. It returns one score for each document, in the order of the documents.
type Reranker interface {
	Rerank(ctx context.Context, call connection.Call, request *RerankRequest) (*RerankResult, error)
}

// Profiler carries out llm.profile for the model of the row of the call.
type Profiler interface {
	GetProfile(ctx context.Context, call connection.Call) (*Profile, error)
}

// ModelLister carries out llm.models: the models of the connection that fit the operations of the query.
type ModelLister interface {
	ListModels(ctx context.Context, call connection.Call, query connection.OptionsQuery) (connection.ListedOptions, error)
}

// ReasoningLister carries out llm.reasoning: the reasoning levels of the model of the row of the call.
type ReasoningLister interface {
	ListReasoningLevels(ctx context.Context, call connection.Call) ([]connection.Option, error)
}

// Roles are the operations of the contract one plugin serves. A nil role is not served.
//
// A role returns connection.NewTryNextError or connection.NewStopError for a call that failed, and
// ctx.Err() for a call that was canceled.
type Roles struct {
	Completer Completer
	Chatter   Chatter
	Embedder  Embedder
	Reranker  Reranker
	Profiler  Profiler
	Models    ModelLister
	Reasoning ReasoningLister
}

// Provide adds the operations of the roles to provider, and returns it. A nil provider is a new one.
func Provide(provider connection.Provider, roles Roles) connection.Provider {
	if provider == nil {
		provider = connection.Provider{}
	}
	if roles.Completer != nil {
		provider[OperationComplete] = serveOnce(func(ctx context.Context, call connection.Call, request *CompleteRequest) (proto.Message, error) {
			return roles.Completer.Complete(ctx, call, request)
		})
	}
	if roles.Embedder != nil {
		provider[OperationEmbed] = serveOnce(func(ctx context.Context, call connection.Call, request *EmbedRequest) (proto.Message, error) {
			return roles.Embedder.Embed(ctx, call, request)
		})
	}
	if roles.Reranker != nil {
		provider[OperationRerank] = serveOnce(func(ctx context.Context, call connection.Call, request *RerankRequest) (proto.Message, error) {
			return roles.Reranker.Rerank(ctx, call, request)
		})
	}
	if roles.Profiler != nil {
		provider[OperationProfile] = serveOnce(func(ctx context.Context, call connection.Call, _ *ProfileRequest) (proto.Message, error) {
			return roles.Profiler.GetProfile(ctx, call)
		})
	}
	if roles.Chatter != nil {
		provider[OperationChat] = serveChat(roles.Chatter)
	}
	if roles.Models != nil {
		provider[OperationModels] = connection.ServeOptions(roles.Models.ListModels)
	}
	if roles.Reasoning != nil {
		provider[OperationReasoning] = connection.ServeOptions(func(ctx context.Context, call connection.Call, _ connection.OptionsQuery) (connection.ListedOptions, error) {
			levels, err := roles.Reasoning.ListReasoningLevels(ctx, call)
			return connection.ListedOptions{Options: levels}, err
		})
	}
	return provider
}

func serveOnce[Q any, P interface {
	*Q
	proto.Message
}](do func(ctx context.Context, call connection.Call, request P) (proto.Message, error)) connection.Operation {
	return func(ctx context.Context, call connection.Call, payload []byte, _ func([]byte) error) ([]byte, error) {
		request := P(new(Q))
		if err := proto.Unmarshal(payload, request); err != nil {
			return nil, connection.NewStopError(connection.ReasonInvalidRequest)
		}
		result, err := do(ctx, call, request)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(result)
	}
}

func serveChat(chatter Chatter) connection.Operation {
	return func(ctx context.Context, call connection.Call, payload []byte, send func([]byte) error) ([]byte, error) {
		request := &ChatRequest{}
		if err := proto.Unmarshal(payload, request); err != nil {
			return nil, connection.NewStopError(connection.ReasonInvalidRequest)
		}
		finish, err := chatter.Chat(ctx, call, request, func(frame *ChatFrame) error {
			data, err := proto.Marshal(frame)
			if err != nil {
				return err
			}
			return send(data)
		})
		if err != nil {
			return nil, err
		}
		return proto.Marshal(finish)
	}
}
