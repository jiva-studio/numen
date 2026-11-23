// Package openaiapi sends the requests of a connection to a service that implements the OpenAI API:
// chat completions, embeddings and the list of models, and reranking in the format of Cohere and Jina.
// It keeps no key and no connection between calls, and sends a request only to an address the domain
// allows.
package openaiapi

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/ports"
)

// Clock gets the current time, to read a Retry-After given as a date.
type Clock interface {
	Now() time.Time
}

// Gateway is the OpenAI API as the use cases see it. A request that gets no answer ends with the end of the
// call where the call has ended, stops for an address the domain refuses, and is unavailable, to try the
// next connection after, for a service that is unreachable or slow. A key sent over plain http goes only
// to a loopback address, so it stays on this machine.
type Gateway struct {
	clients       clients
	clock         Clock
	reportFailure func(context.Context, string, string, int)
	tracer        Tracer
}

// SetFailureReporter sets the receiver of HTTP failure classifications.
func (g *Gateway) SetFailureReporter(report func(context.Context, string, string, int)) {
	g.reportFailure = report
}

func (g *Gateway) report(ctx context.Context, operation, failure string, status int) {
	if g.reportFailure != nil && ctx.Err() == nil {
		g.reportFailure(ctx, operation, failure, status)
	}
}

var _ ports.Gateway = (*Gateway)(nil)

// New returns a Gateway that reads a date of the services against clock.
func New(clock Clock) *Gateway {
	return &Gateway{clients: newClients(), clock: clock, reportFailure: nil, tracer: nil}
}

// Close closes the idle connections.
func (g *Gateway) Close() {
	g.clients.close()
}

type reasoningBody struct {
	Effort string `json:"effort"`
}

// answerBounds is the part of a request body that bounds the answer: its tokens and its reasoning, in the
// fields the type of the service reads.
type answerBounds struct {
	MaxTokens           int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens int            `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
	Reasoning           *reasoningBody `json:"reasoning,omitempty"`
}

func newAnswerBounds(call connection.Call, maxTokens int) answerBounds {
	bounds := answerBounds{MaxTokens: 0, MaxCompletionTokens: 0, ReasoningEffort: "", Reasoning: nil}
	bounds.MaxTokens, bounds.MaxCompletionTokens = splitTokenLimit(call.Type, maxTokens)
	bounds.ReasoningEffort, bounds.Reasoning = splitReasoning(call.Type, call.Options[llm.OptionReasoning])
	return bounds
}

// Complete asks the chat completions endpoint of the service for an answer. The answer is read as a
// stream, so the limit is how long the service may stay silent, as for Chat.
func (g *Gateway) Complete(ctx context.Context, call connection.Call, limit time.Duration, request *llm.CompleteRequest) (*llm.CompleteResult, error) {
	messages := make([]*llm.ChatMessage, len(request.GetMessages()))
	for i, message := range request.GetMessages() {
		messages[i] = &llm.ChatMessage{Role: message.GetRole(), Text: message.GetText()}
	}
	var text strings.Builder
	_, err := g.Chat(ctx, call, limit,
		&llm.ChatRequest{Messages: messages, Temperature: request.Temperature, MaxTokens: request.GetMaxTokens()},
		func(frame *llm.ChatFrame) error {
			text.WriteString(frame.GetTextDelta())
			return nil
		})
	if err != nil {
		return nil, err
	}
	return &llm.CompleteResult{Text: text.String()}, nil
}

func splitTokenLimit(typeID string, limit int) (maxTokens, maxCompletionTokens int) {
	if typeID == servicetype.OpenAI {
		return 0, limit
	}
	return limit, 0
}

func splitReasoning(typeID, level string) (effort string, object *reasoningBody) {
	if typeID == servicetype.OpenRouter && level != "" {
		return "", &reasoningBody{Effort: level}
	}
	return level, nil
}

type embedBody struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedAnswer struct {
	Data []struct {
		Index     *int      `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// Embed asks the embeddings endpoint of the service for one vector for each text.
func (g *Gateway) Embed(ctx context.Context, call connection.Call, limit time.Duration, texts []string) ([][]float32, error) {
	var answer embedAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodPost, "/embeddings"}, embedBody{Model: call.Options[llm.OptionModel], Input: texts}, &answer); err != nil {
		return nil, err
	}
	if len(answer.Data) != len(texts) {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	vectors := make([][]float32, len(texts))
	for position, one := range answer.Data {
		index := position
		if one.Index != nil {
			index = *one.Index
		}
		if index < 0 || index >= len(vectors) || vectors[index] != nil || len(one.Embedding) == 0 {
			return nil, connection.NewStopError(connection.ReasonUnavailable)
		}
		vectors[index] = one.Embedding
	}
	return vectors, nil
}

type modelsAnswer struct {
	Data []struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		Architecture struct {
			OutputModalities []string `json:"output_modalities"`
		} `json:"architecture"`
	} `json:"data"`
}

// ListModels lists the models of the service that fit the operations of a setting of rows. OpenRouter
// labels its models, so the list holds its embedding models (/embeddings/models) for a setting used for
// llm.embed alone, its rerank models (/models?output_modalities=rerank) for a setting used for llm.rerank
// alone, its models of text for a setting used for llm.complete or llm.chat, and none for any other; the
// other services give no label, and every model is listed. A model with no declared output modalities counts
// as a model of text, because OpenRouter leaves the list out for some.
func (g *Gateway) ListModels(ctx context.Context, call connection.Call, limit time.Duration, operations []string) ([]connection.Option, error) {
	path, isKept, isListed := getModelsQuery(call.Type, operations)
	if !isListed {
		return []connection.Option{}, nil
	}
	var answer modelsAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodGet, path}, nil, &answer); err != nil {
		return nil, err
	}
	models := make([]connection.Option, 0, len(answer.Data))
	for _, one := range answer.Data {
		if isKept(one.Architecture.OutputModalities) {
			models = append(models, connection.Option{Value: one.ID, Title: cmp.Or(one.Name, one.ID)})
		}
	}
	slices.SortFunc(models, func(a, b connection.Option) int { return strings.Compare(a.Value, b.Value) })
	return models, nil
}

func getModelsQuery(typeID string, operations []string) (path string, isKept func([]string) bool, isListed bool) {
	if typeID != servicetype.OpenRouter {
		return "/models", isAnyModel, true
	}
	switch {
	case slices.Contains(operations, llm.OperationComplete) || slices.Contains(operations, llm.OperationChat):
		return "/models", canGenerateText, true
	case slices.Contains(operations, llm.OperationEmbed):
		return "/embeddings/models", isAnyModel, true
	case slices.Contains(operations, llm.OperationRerank):
		return "/models?output_modalities=rerank", canRerank, true
	default:
		return "", nil, false
	}
}

func isAnyModel([]string) bool { return true }

func canGenerateText(modalities []string) bool {
	return len(modalities) == 0 || slices.Contains(modalities, "text")
}

func canRerank(modalities []string) bool {
	return slices.Contains(modalities, "rerank")
}
