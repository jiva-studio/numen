package completion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/usecase/completion"
)

type gateway struct {
	limit time.Duration
	err   error
	calls int
}

func (g *gateway) Complete(_ context.Context, _ connection.Call, limit time.Duration, _ *llm.CompleteRequest) (*llm.CompleteResult, error) {
	g.calls++
	g.limit = limit
	return &llm.CompleteResult{Text: "answer"}, g.err
}

func (g *gateway) Chat(
	_ context.Context, _ connection.Call, limit time.Duration, _ *llm.ChatRequest, send func(*llm.ChatFrame) error,
) (*llm.ChatFinish, error) {
	g.calls++
	g.limit = limit
	if g.err != nil {
		return nil, g.err
	}
	return &llm.ChatFinish{Reason: llm.FinishReason_FINISH_REASON_STOP}, send(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: "hi"}})
}

func (g *gateway) Embed(context.Context, connection.Call, time.Duration, []string) ([][]float32, error) {
	return nil, nil
}

func (g *gateway) ListModels(context.Context, connection.Call, time.Duration, []string) ([]connection.Option, error) {
	return nil, nil
}

func ignoreFrame(*llm.ChatFrame) error { return nil }

func newCompletion(t *testing.T, g *gateway) *completion.Completion {
	t.Helper()
	subject, err := completion.New(g, &limits.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	return subject
}

func call(typeID, model, level string) connection.Call {
	return connection.Call{
		Type: typeID, Options: map[string]string{llm.OptionModel: model, llm.OptionReasoning: level}, Fields: map[string]string{"baseUrl": "https://api.test/v1"},
	}
}

func TestACompletionIsAskedOfTheServiceWithTheLimitOfTheOperation(t *testing.T) {
	g := &gateway{}

	result, err := newCompletion(t, g).Complete(t.Context(), call("openai", "gpt-4o", ""), &llm.CompleteRequest{})

	if err != nil || result.GetText() != "answer" || g.limit != limits.DefaultCompleteLimit {
		t.Errorf("result = %v, err = %v, limit = %v", result, err, g.limit)
	}
}

func TestALevelOfReasoningOutsideTheLevelsOfTheModelIsAStopBeforeAnyRequest(t *testing.T) {
	g := &gateway{}
	subject := newCompletion(t, g)

	_, completeErr := subject.Complete(t.Context(), call("openai", "gpt-4o", "high"), &llm.CompleteRequest{})
	_, chatErr := subject.Chat(t.Context(), call("openai", "gpt-4o", "high"), &llm.ChatRequest{}, ignoreFrame)

	for _, err := range []error{completeErr, chatErr} {
		if err == nil {
			t.Errorf("err = %v, want a stop", err)
		}
	}
	if g.calls != 0 {
		t.Errorf("the service was asked %d times, want none", g.calls)
	}
}

func TestAChatPassesItsFramesAndItsFinishOn(t *testing.T) {
	g := &gateway{}
	var events int

	finish, err := newCompletion(t, g).Chat(t.Context(), call("openrouter", "m", "low"), &llm.ChatRequest{}, func(*llm.ChatFrame) error {
		events++
		return nil
	})

	if err != nil || events != 1 || finish.GetReason() != llm.FinishReason_FINISH_REASON_STOP {
		t.Errorf("err = %v, events = %d", err, events)
	}
}

func TestAFailureOfTheServiceIsReturnedAsItIs(t *testing.T) {
	failure := connection.NewStopError(connection.ReasonInvalidRequest)
	g := &gateway{err: failure}
	subject := newCompletion(t, g)

	_, completeErr := subject.Complete(t.Context(), call("openai", "m", ""), &llm.CompleteRequest{})
	_, chatErr := subject.Chat(t.Context(), call("openai", "m", ""), &llm.ChatRequest{}, ignoreFrame)

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
