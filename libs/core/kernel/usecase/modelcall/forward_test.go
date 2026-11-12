package modelcall_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

func TestForwardedCallsReachTheCallsSetLater(t *testing.T) {
	s := newSetup(t, "complete")
	var current connectionport.ModelCalls
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return current })

	_, before := forwarded.Complete(t.Context(), consumerID, modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)
	current = s.calls
	result, after := forwarded.Complete(t.Context(), consumerID, modelcalldomain.CompleteRequest{}, modelcalldomain.PriorityBackground)

	if !errors.Is(before, connection.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || result.Text != "answer from "+addressOne {
		t.Errorf("after the calls are set: %v, %v", result, after)
	}
}

func TestForwardedCallsWithNoCallsAnswerNoConnectionToEveryOperation(t *testing.T) {
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return nil })

	_, embedErr := forwarded.Embed(t.Context(), consumerID, []string{"a"}, modelcalldomain.PriorityBackground)
	_, profileErr := forwarded.GetEmbeddingProfile(t.Context(), consumerID)

	if !errors.Is(embedErr, connection.ErrNoConnection) || !errors.Is(profileErr, connection.ErrNoConnection) {
		t.Errorf("errors = %v, %v", embedErr, profileErr)
	}
}

func TestForwardedChatReachesTheCallsSetLaterAndFindsNoConnectionBefore(t *testing.T) {
	s := newSetup(t, "chat")
	var current connectionport.ModelCalls
	forwarded := modelcall.NewForwardedCalls(func() connectionport.ModelCalls { return current })
	send := func(modelcalldomain.ChatEvent) error { return nil }

	before := forwarded.Chat(t.Context(), consumerID, modelcalldomain.ChatRequest{}, modelcalldomain.PriorityBackground, send)
	current = s.calls
	var text string
	after := forwarded.Chat(t.Context(), consumerID, modelcalldomain.ChatRequest{}, modelcalldomain.PriorityBackground,
		func(event modelcalldomain.ChatEvent) error {
			text += event.TextDelta
			return nil
		})

	if !errors.Is(before, connection.ErrNoConnection) {
		t.Errorf("before the calls are set: err = %v, want no connection", before)
	}
	if after != nil || text != "answer from "+addressOne {
		t.Errorf("after the calls are set: %q, %v", text, after)
	}
}
