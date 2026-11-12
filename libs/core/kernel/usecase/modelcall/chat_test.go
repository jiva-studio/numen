package modelcall_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	modelcalldomain "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

func (s *setup) chat(t *testing.T) ([]modelcalldomain.ChatEvent, error) {
	t.Helper()
	return s.chatAs(t.Context(), consumerID, modelcalldomain.PriorityBackground)
}

func (s *setup) chatAs(ctx context.Context, plugin string, priority modelcalldomain.Priority) ([]modelcalldomain.ChatEvent, error) {
	var events []modelcalldomain.ChatEvent
	err := s.calls.Chat(ctx, plugin, modelcalldomain.ChatRequest{
		Messages: []modelcalldomain.ChatMessage{{Role: "user", Text: "list the notes"}},
		Tools:    []modelcalldomain.ToolDefinition{{Name: "list_notes", Parameters: `{"type":"object"}`}},
	}, priority, func(event modelcalldomain.ChatEvent) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

func texts(events []modelcalldomain.ChatEvent) string {
	var text strings.Builder
	for _, one := range events {
		text.WriteString(one.TextDelta)
	}
	return text.String()
}

func TestAChatTurnIsStreamedFromTheFirstRowThatAnswers(t *testing.T) {
	s := newSetup(t, "chat")
	stop := modelcalldomain.ChatFinishStop
	s.world.Provider.ScriptChat(addressOne, nil,
		modelcalldomain.ChatEvent{TextDelta: "Let me "}, modelcalldomain.ChatEvent{TextDelta: "look."},
		modelcalldomain.ChatEvent{ToolCall: &modelcalldomain.ToolCall{ID: "c1", Name: "list_notes", Arguments: "{}"}},
		modelcalldomain.ChatEvent{Finish: &stop},
	)

	events, err := s.chat(t)

	if err != nil || len(events) != 4 || texts(events) != "Let me look." || events[2].ToolCall.Name != "list_notes" || events[3].Finish == nil {
		t.Fatalf("Chat = %v, %v", events, err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the second row was called though the first answered")
	}
}

func TestAChatCallCarriesTheMessagesToolsAndTheSnapshotOfTheRow(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Rows.Set(consumerID, "chat", connection.Row{ConnectionID: s.one.ID, Model: "model-one", Reasoning: "low"})

	if _, err := s.chat(t); err != nil {
		t.Fatal(err)
	}

	got := s.world.Provider.CallsTo(addressOne)[0]
	if got.Call.Model != "model-one" || got.Call.Reasoning != "low" || got.Call.Secrets["apiKey"] != keyOne {
		t.Errorf("call = %+v", got.Call)
	}
	if len(got.Chat.Messages) != 1 || got.Chat.Messages[0].Text != "list the notes" || got.Chat.Tools[0].Name != "list_notes" {
		t.Errorf("request = %+v", got.Chat)
	}
}

func TestAFailureBeforeTheFirstEventGoesToTheNextRow(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.Fail(addressOne, tryNext(connection.FailureReasonRateLimited, 0))

	events, err := s.chat(t)

	if err != nil || texts(events) != "answer from "+addressTwo {
		t.Fatalf("Chat = %v, %v", events, err)
	}
	if got := len(s.world.Provider.CallsTo(addressOne)); got != 1 {
		t.Errorf("a rate limit was tried %d times, want once", got)
	}
}

func TestAProviderRestartedBeforeTheFirstEventIsStartedAgainForTheRetry(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.Fail(addressOne, errors.New("rpc error: connection reset"))

	events, err := s.chat(t)

	if err != nil || texts(events) != "answer from "+addressOne {
		t.Fatalf("Chat = %v, %v", events, err)
	}
	if got := len(s.world.Source.Opened()); got != 2 {
		t.Errorf("providers opened = %d, want the retry to open it again", got)
	}
}

func TestAFailureAfterTheFirstEventIsPassedOnAndNothingIsRepeated(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.ScriptChat(addressOne, tryNext(connection.FailureReasonUnreachable, 0),
		modelcalldomain.ChatEvent{TextDelta: "partial"},
		modelcalldomain.ChatEvent{ToolCall: &modelcalldomain.ToolCall{ID: "c1", Name: "list_notes", Arguments: "{}"}},
	)

	events, err := s.chat(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) || !slices.Equal(failed.Failures, []connection.Failure{{ConnectionName: "First", Reason: "unreachable"}}) {
		t.Fatalf("err = %v, want a failure naming First and its reason", err)
	}
	if len(events) != 2 || events[0].TextDelta != "partial" {
		t.Errorf("events = %v, want the text and the tool call kept", events)
	}
	if len(s.world.Provider.CallsTo(addressOne)) != 1 || len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Errorf("calls to One = %d, to Two = %d, want one and none", len(s.world.Provider.CallsTo(addressOne)), len(s.world.Provider.CallsTo(addressTwo)))
	}
}

func TestAPlainErrorAfterTheFirstEventCountsAsUnreachable(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.ScriptChat(addressOne, errors.New("rpc error: connection reset by Bearer "+keyOne),
		modelcalldomain.ChatEvent{TextDelta: "partial"})

	_, err := s.chat(t)

	if err == nil || err.Error() != "First: unreachable" {
		t.Errorf("err = %v", err)
	}
}

func TestAStopAfterTheFirstEventNamesTheConnection(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.ScriptChat(addressOne, stop(connection.FailureReasonBadRequest), modelcalldomain.ChatEvent{TextDelta: "partial"})

	_, err := s.chat(t)

	var failed *connection.AllFailedError
	if !errors.As(err, &failed) || failed.Failures[0].Reason != "bad request" {
		t.Errorf("err = %v", err)
	}
}

func TestAStopBeforeTheFirstEventEndsTheCallAndNamesTheConnection(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.Fail(addressOne, stop(connection.FailureReasonBadRequest))

	_, err := s.chat(t)

	var rejected *connection.RejectedError
	if !errors.As(err, &rejected) || rejected.ConnectionName != "First" || rejected.Reason != "bad request" {
		t.Fatalf("err = %v, want a rejection by First", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the call went on to the next row after a stop")
	}
}

func TestWhenEveryRowFailsBeforeAnyEventTheErrorNamesEachConnection(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.FailAlways(addressOne, tryNext(connection.FailureReasonPaymentRequired, 0))
	s.world.Provider.FailAlways(addressTwo, tryNext(connection.FailureReasonRateLimited, 0))

	events, err := s.chat(t)

	if len(events) != 0 || err == nil || err.Error() != "First: payment required; Second: rate limited" {
		t.Errorf("events = %v, err = %v", events, err)
	}
}

func TestAFailureAfterTheFirstEventSetsTheConnectionAside(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.ScriptChat(addressOne, tryNext(connection.FailureReasonRateLimited, 0), modelcalldomain.ChatEvent{TextDelta: "partial"})
	if _, err := s.chat(t); err == nil {
		t.Fatal("the first turn did not fail")
	}

	events, err := s.chat(t)

	if err != nil || texts(events) != "answer from "+addressTwo {
		t.Errorf("Chat = %v, %v, want the second row after the first was set aside", events, err)
	}
}

func TestAChatTurnOfAPluginWithNoChatRowGetsNoConnection(t *testing.T) {
	s := newSetup(t, "complete")

	_, err := s.chat(t)

	if !errors.Is(err, connection.ErrNoConnection) {
		t.Errorf("err = %v, want ErrNoConnection", err)
	}
}

func TestAChatTurnSkipsATypeThatCannotChat(t *testing.T) {
	s := newSetup(t, "chat")
	completer := s.world.Add(t, modelcalltest.ProviderPluginID+"/completer", "Completer", "https://completer.test/v1", "")
	s.world.Rows.Set(consumerID, "chat", modelcalltest.Row(completer, "m"), modelcalltest.Row(s.one, "m"))

	events, err := s.chat(t)

	if err != nil || texts(events) != "answer from "+addressOne {
		t.Errorf("Chat = %v, %v", events, err)
	}
	if len(s.world.Provider.CallsTo("https://completer.test/v1")) != 0 {
		t.Error("a type without the operation chat was called")
	}
}

func TestAnErrorOfTheConsumerEndsTheTurnAndIsReturnedAsItIs(t *testing.T) {
	s := newSetup(t, "chat")
	consumerFailure := errors.New("the consumer went away")

	err := s.calls.Chat(t.Context(), consumerID, modelcalldomain.ChatRequest{}, modelcalldomain.PriorityBackground,
		func(modelcalldomain.ChatEvent) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the consumer", err)
	}
	if len(s.world.Provider.CallsTo(addressTwo)) != 0 {
		t.Error("the next row was called after the consumer went away")
	}
}

func TestACanceledChatReturnsTheErrorOfTheContextAndFreesTheLease(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.Block(addressOne)
	ctx, cancel := context.WithCancel(t.Context())
	finish := make(chan error, 1)
	go func() {
		_, err := s.chatAs(ctx, consumerID, modelcalldomain.PriorityBackground)
		finish <- err
	}()
	<-s.world.Provider.Entered()

	cancel()

	select {
	case err := <-finish:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a canceled chat did not return")
	}
	if opened, released := len(s.world.Source.Opened()), s.world.Source.Released(); opened != released {
		t.Errorf("opened %d providers and released %d", opened, released)
	}
}

func TestNothingAChatConsumerReceivesContainsAKey(t *testing.T) {
	s := newSetup(t, "chat")
	s.world.Provider.ScriptChat(addressOne, errors.New("HTTP 401: Authorization: Bearer "+keyOne), modelcalldomain.ChatEvent{TextDelta: "partial"})

	events, err := s.chat(t)

	received := texts(events)
	if err != nil {
		received += err.Error()
	}
	if strings.Contains(received, "sk-") {
		t.Errorf("the consumer received a key: %q", received)
	}
}

func TestTheSlotOfAConnectionIsHeldForTheLengthOfTheStream(t *testing.T) {
	s := newQueueSetupFor(t, "chat", 1)
	release := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		first <- s.calls.Chat(t.Context(), consumerID, modelcalldomain.ChatRequest{}, modelcalldomain.PriorityBackground,
			func(modelcalldomain.ChatEvent) error {
				<-release
				return nil
			})
	}()
	s.awaitEntered(t)
	second := make(chan error, 1)
	go func() {
		second <- s.calls.Chat(t.Context(), otherConsumerID, modelcalldomain.ChatRequest{}, modelcalldomain.PriorityBackground,
			func(modelcalldomain.ChatEvent) error { return nil })
	}()

	select {
	case <-s.world.Provider.Entered():
		t.Fatal("a second turn reached the provider while the first stream was open on a connection of one slot")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)

	if err := awaitDone(t, first); err != nil {
		t.Errorf("first turn: %v", err)
	}
	if err := awaitDone(t, second); err != nil {
		t.Errorf("second turn: %v", err)
	}
}

func newQueueSetupFor(t *testing.T, operation string, parallelRequests int) *queueSetup {
	t.Helper()
	w := modelcalltest.New()
	entry := w.AddLimited(t, "Limited", addressLimited, parallelRequests)
	for _, plugin := range []string{consumerID, otherConsumerID} {
		w.Rows.Set(plugin, operation, modelcalltest.Row(entry, "m"))
	}
	return &queueSetup{
		world: w, entry: entry,
		calls: modelcall.NewCalls(w.Resolve, w.ReadSecrets, w.Rows, w.Source, w.Clock, w.Waits.Wait, w.Notices.ReportSetAside),
	}
}
