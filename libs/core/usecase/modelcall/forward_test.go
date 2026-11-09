package modelcall_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/usecase/modelcall"
)

func TestForwardedCallsReachTheCallsSetLater(t *testing.T) {
	s := newSetup(t, "complete")
	var current connectionport.ModelCalls
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return current })

	_, before := forwarded.Complete(t.Context(), consumerID, connectionport.CompleteRequest{}, connectionport.PriorityBackground)
	current = s.calls
	result, after := forwarded.Complete(t.Context(), consumerID, connectionport.CompleteRequest{}, connectionport.PriorityBackground)

	if !errors.Is(before, connectionport.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || result.Text != "answer from "+addressOne {
		t.Errorf("after the calls are set: %v, %v", result, after)
	}
}

func TestForwardedCallsWithNoCallsAnswerNoConnectionToEveryOperation(t *testing.T) {
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return nil })

	_, embedErr := forwarded.Embed(t.Context(), consumerID, []string{"a"}, connectionport.PriorityBackground)
	_, profileErr := forwarded.GetEmbeddingProfile(t.Context(), consumerID)

	if !errors.Is(embedErr, connectionport.ErrNoConnection) || !errors.Is(profileErr, connectionport.ErrNoConnection) {
		t.Errorf("errors = %v, %v", embedErr, profileErr)
	}
}

func TestForwardedChatReachesTheCallsSetLaterAndFindsNoConnectionBefore(t *testing.T) {
	s := newSetup(t, "chat")
	var current connectionport.ModelCalls
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return current })
	send := func(connectionport.ChatEvent) error { return nil }

	before := forwarded.Chat(t.Context(), consumerID, connectionport.ChatRequest{}, connectionport.PriorityBackground, send)
	current = s.calls
	var text string
	after := forwarded.Chat(t.Context(), consumerID, connectionport.ChatRequest{}, connectionport.PriorityBackground,
		func(event connectionport.ChatEvent) error {
			text += event.TextDelta
			return nil
		})

	if !errors.Is(before, connectionport.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || text != "answer from "+addressOne {
		t.Errorf("after the calls are set: %q, %v", text, after)
	}
}
