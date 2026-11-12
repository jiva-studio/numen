package llmtest_test

import (
	"context"
	"errors"
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

func TestTheCallerEndsWithTheErrorOfTheOperation(t *testing.T) {
	stopped := connection.NewStopError(connection.ReasonRejectedKey)

	_, err := llm.Complete(t.Context(), llmtest.NewCaller(newProvider(stopped), connection.Call{}), &llm.CompleteRequest{}, connection.PriorityBackground)

	if !errors.Is(err, stopped) {
		t.Errorf("Complete = %v, want the error of the operation", err)
	}
}
