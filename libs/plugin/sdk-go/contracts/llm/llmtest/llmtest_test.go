package llmtest_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
)

// echo completes with the model of the row and chats the words of the last message, one frame each.
type echo struct{ err error }

func (e echo) Complete(_ context.Context, call connection.Call, _ *llm.CompleteRequest) (*llm.CompleteResult, error) {
	return &llm.CompleteResult{Text: call.Options[llm.OptionModel]}, e.err
}

func (e echo) Chat(_ context.Context, _ connection.Call, _ *llm.ChatRequest, send func(*llm.ChatFrame) error) (*llm.ChatFinish, error) {
	for _, word := range []string{"tide", "turns"} {
		if err := send(&llm.ChatFrame{Event: &llm.ChatFrame_TextDelta{TextDelta: word}}); err != nil {
			return nil, err
		}
	}
	return &llm.ChatFinish{}, e.err
}

func newProvider(err error) connection.Provider {
	return llm.Provide(nil, llm.Roles{Completer: echo{err: err}, Chatter: echo{err: err}})
}

func TestCompleteAsksTheOperationOfTheProvider(t *testing.T) {
	call := connection.Call{Options: map[string]string{llm.OptionModel: "harbor-1"}}

	result, err := llmtest.Complete(t.Context(), newProvider(nil), call, &llm.CompleteRequest{})

	if err != nil || result.GetText() != "harbor-1" {
		t.Errorf("Complete = %v, %v, want the model of the row", result, err)
	}
}

func TestTheCallerAnswersTheClientOfTheContract(t *testing.T) {
	stream, err := llm.Chat(t.Context(), llmtest.NewCaller(newProvider(nil), connection.Call{}), &llm.ChatRequest{}, connection.PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	var words []string
	for stream.Next() {
		words = append(words, stream.Frame().GetTextDelta())
	}
	if err := stream.Err(); err != nil || len(words) != 2 || words[0] != "tide" || words[1] != "turns" || stream.Finish() == nil {
		t.Errorf("frames %v, err %v, want two words and a finish", words, err)
	}
}

func TestTheCallerEndsWithTheErrorsOfTheCore(t *testing.T) {
	call := connection.Call{ConnectionID: "harbor"}
	cases := map[string]struct {
		provider connection.Provider
		want     error
	}{
		"a stop": {
			provider: newProvider(connection.NewStopError(connection.ReasonRejectedKey)),
			want:     &connection.StoppedError{ConnectionName: "harbor", Reason: "key rejected"},
		},
		"an error that is no failure kind": {
			provider: newProvider(errors.New("the socket closed")),
			want:     &connection.StoppedError{ConnectionName: "harbor", Reason: "unavailable"},
		},
		"a try-next": {
			provider: newProvider(connection.NewTryNextError(connection.ReasonRateLimited, 0)),
			want:     &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "harbor", Reason: "rate limited"}}},
		},
		"an operation the provider does not serve": {
			provider: connection.Provider{},
			want: &connection.AllFailedError{Failures: []connection.Failure{
				{ConnectionName: "harbor", Reason: "the type does not serve " + llm.OperationComplete},
			}},
		},
	}
	for name, one := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := llm.Complete(t.Context(), llmtest.NewCaller(one.provider, call), &llm.CompleteRequest{}, connection.PriorityBackground)

			if err == nil || err.Error() != one.want.Error() || reflect.TypeOf(err) != reflect.TypeOf(one.want) {
				t.Errorf("Complete = %v, want %v", err, one.want)
			}
		})
	}
}

func TestAStopAfterTheFirstFrameEndsTheCallerWithEveryRowFailed(t *testing.T) {
	provider := newProvider(connection.NewStopError(connection.ReasonInvalidRequest))
	stream, err := llm.Chat(t.Context(), llmtest.NewCaller(provider, connection.Call{ConnectionID: "harbor"}), &llm.ChatRequest{}, connection.PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	var frames int
	for stream.Next() {
		frames++
	}

	all, isAllFailed := errors.AsType[*connection.AllFailedError](stream.Err())
	if frames != 2 || !isAllFailed || len(all.Failures) != 1 || all.Failures[0].Reason != "invalid request" {
		t.Errorf("frames %d, err %v, want two frames and every row failed", frames, stream.Err())
	}
}

func TestACanceledCallEndsTheCallerWithTheErrorOfTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	provider := connection.Provider{llm.OperationComplete: func(ctx context.Context, _ connection.Call, _ []byte, _ func([]byte) error) ([]byte, error) {
		cancel()
		return nil, ctx.Err()
	}}

	_, err := llm.Complete(ctx, llmtest.NewCaller(provider, connection.Call{}), &llm.CompleteRequest{}, connection.PriorityBackground)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Complete = %v, want the cancellation", err)
	}
}

func TestAnOperationWithoutFramesThatSendsOneGetsAnError(t *testing.T) {
	sendErr := errors.New("no send")
	provider := connection.Provider{llm.OperationComplete: func(_ context.Context, _ connection.Call, _ []byte, send func([]byte) error) ([]byte, error) {
		if err := send([]byte("wave")); err != nil {
			return nil, err
		}
		return nil, sendErr
	}}

	_, err := llmtest.Complete(t.Context(), provider, connection.Call{}, &llm.CompleteRequest{})

	if err == nil || errors.Is(err, sendErr) {
		t.Errorf("Complete = %v, want the error of the send", err)
	}
}
