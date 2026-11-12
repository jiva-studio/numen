package completion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
)

type gateway struct {
	limit time.Duration
	err   error
	calls int
}

func (g *gateway) Complete(_ context.Context, _ connection.Call, limit time.Duration, _ connection.ProviderCompleteRequest) (connection.CompleteResult, error) {
	g.calls++
	g.limit = limit
	return connection.CompleteResult{Text: "answer"}, g.err
}

func (g *gateway) Chat(_ context.Context, _ connection.Call, limit time.Duration, _ connection.ProviderChatRequest, send func(connection.ChatEvent) error) error {
	g.calls++
	g.limit = limit
	if g.err != nil {
		return g.err
	}
	return send(connection.ChatEvent{})
}

func (g *gateway) Embed(context.Context, connection.Call, time.Duration, []string) ([][]float32, error) {
	return nil, nil
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, string) ([]connection.Model, error) {
	return nil, nil
}

func newCompletion(t *testing.T, g *gateway) *completion.Completion {
	t.Helper()
	subject, err := completion.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func call(typeID, model, level string) connection.Call {
	return connection.Call{Type: typeID, Model: model, Reasoning: level, Fields: map[string]string{"baseUrl": "https://api.test/v1"}}
}

func TestACompletionIsAskedOfTheServiceWithTheLimitOfTheOperation(t *testing.T) {
	g := &gateway{}

	result, err := newCompletion(t, g).Complete(t.Context(), call("openai", "gpt-4o", ""), connection.ProviderCompleteRequest{})

	if err != nil || result.Text != "answer" || g.limit != limits.DefaultCompleteLimit {
		t.Errorf("result = %v, err = %v, limit = %v", result, err, g.limit)
	}
}

func TestALevelOfReasoningOutsideTheLevelsOfTheModelIsAStopBeforeAnyRequest(t *testing.T) {
	g := &gateway{}
	subject := newCompletion(t, g)

	_, completeErr := subject.Complete(t.Context(), call("openai", "gpt-4o", "high"), connection.ProviderCompleteRequest{})
	chatErr := subject.Chat(t.Context(), call("openai", "gpt-4o", "high"), connection.ProviderChatRequest{}, func(connection.ChatEvent) error { return nil })

	for _, err := range []error{completeErr, chatErr} {
		if err == nil {
			t.Errorf("err = %v, want a stop", err)
		}
	}
	if g.calls != 0 {
		t.Errorf("the service was asked %d times, want none", g.calls)
	}
}

func TestAChatPassesItsEventsOn(t *testing.T) {
	g := &gateway{}
	var events int

	err := newCompletion(t, g).Chat(t.Context(), call("openrouter", "m", "low"), connection.ProviderChatRequest{}, func(connection.ChatEvent) error {
		events++
		return nil
	})

	if err != nil || events != 1 {
		t.Errorf("err = %v, events = %d", err, events)
	}
}

func TestAFailureOfTheServiceIsReturnedAsItIs(t *testing.T) {
	failure := connection.NewStopError(connection.ReasonBadRequest)
	g := &gateway{err: failure}
	subject := newCompletion(t, g)

	_, completeErr := subject.Complete(t.Context(), call("openai", "m", ""), connection.ProviderCompleteRequest{})
	chatErr := subject.Chat(t.Context(), call("openai", "m", ""), connection.ProviderChatRequest{}, func(connection.ChatEvent) error { return nil })

	if !errors.Is(completeErr, failure) || !errors.Is(chatErr, failure) {
		t.Errorf("errors = %v, %v, want the failure of the service", completeErr, chatErr)
	}
}

func TestACompletionNeedsAGatewayAndLimits(t *testing.T) {
	if _, err := completion.New(nil, &limits.Limits{}); err == nil {
		t.Error("a completion was made with no gateway")
	}
	if _, err := completion.New(&gateway{}, nil); err == nil {
		t.Error("a completion was made with no limits")
	}
}
