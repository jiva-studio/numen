package hostendpoint

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
	modelcalls "github.com/jiva-studio/numen/libs/core/kernel/usecase/modelcall"
)

func readChat(t *testing.T, stream *connect.ServerStreamForClient[v1.HostModelServiceChatResponse]) []*v1.HostModelServiceChatResponse {
	t.Helper()
	var got []*v1.HostModelServiceChatResponse
	for stream.Receive() {
		got = append(got, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream: %v", err)
	}
	return got
}

func TestChatStreamsTheEventsOfTheTurnForThePluginTheTokenNames(t *testing.T) {
	finish := modelcall.ChatFinishToolCalls
	calls := &scriptedCalls{events: []modelcall.ChatEvent{
		{TextDelta: "Let me "},
		{TextDelta: "look."},
		{ToolCall: &modelcall.ToolCall{ID: "c1", Name: "list_notes", Arguments: `{"limit":3}`}},
		{Finish: &finish},
	}}
	client, _ := newModelClient(t, calls, "token-b")
	temperature := 0.4

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{
		Messages: []*v1.HostModelChatMessage{
			{Role: "user", Text: "list"},
			{Role: "assistant", ToolCalls: []*v1.HostModelToolCall{{Id: "c0", Name: "list_notes", ArgumentsJson: "{}"}}},
			{Role: "tool", Text: "a, b", ToolCallId: "c0"},
		},
		Tools:       []*v1.HostModelToolDefinition{{Name: "list_notes", Description: "Lists.", ParametersJson: `{"type":"object"}`}},
		Temperature: &temperature, MaxTokens: 33, Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 4 || got[0].GetTextDelta() != "Let me " || got[1].GetTextDelta() != "look." ||
		got[2].GetToolCall().GetId() != "c1" || got[2].GetToolCall().GetArgumentsJson() != `{"limit":3}` ||
		got[3].GetFinish().GetReason() != v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS {
		t.Errorf("events = %v", got)
	}
	if len(calls.callers) != 1 || calls.callers[0] != "md.numen.b" {
		t.Errorf("callers = %v, want md.numen.b: the plugin of the token", calls.callers)
	}
	request := calls.chatRequest
	if len(request.Messages) != 3 || request.Messages[1].ToolCalls[0] != (modelcall.ToolCall{ID: "c0", Name: "list_notes", Arguments: "{}"}) ||
		request.Messages[2].ToolCallID != "c0" || request.Tools[0] != (modelcall.ToolDefinition{Name: "list_notes", Description: "Lists.", Parameters: `{"type":"object"}`}) ||
		*request.Temperature != 0.4 || request.MaxTokens != 33 || calls.priority != modelcall.PriorityInteractive {
		t.Errorf("request = %+v, priority = %v", request, calls.priority)
	}
}

func TestAChatWithoutAKnownTokenIsUnauthenticated(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{}, "no-such-token")
	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
	if err == nil {
		stream.Receive()
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want unauthenticated", connect.CodeOf(err))
	}
}

func TestAChatFailureBeforeTheFirstEventIsAServedErrorThatNamesEachConnection(t *testing.T) {
	failure := &connection.AllFailedError{Failures: []connection.Failure{
		{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
	}}
	client, _ := newModelClient(t, &scriptedCalls{err: failure}, "token-a")

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 1 || got[0].GetFailure().GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED ||
		len(got[0].GetFailure().GetFailures()) != 2 || got[0].GetFailure().GetFailures()[1].GetReason() != "unreachable" ||
		got[0].GetFailure().GetDetail() != "OpenRouter: payment required; Ollama: unreachable" {
		t.Errorf("events = %v", got)
	}
}

func TestAChatFailureAfterTheFirstEventKeepsTheTextAndEndsTheStreamWithTheError(t *testing.T) {
	calls := &scriptedCalls{
		events: []modelcall.ChatEvent{{TextDelta: "partial"}},
		err:    &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "Ollama", Reason: "unreachable"}}},
	}
	client, _ := newModelClient(t, calls, "token-a")

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 2 || got[0].GetTextDelta() != "partial" || got[1].GetFailure().GetFailures()[0].GetConnectionName() != "Ollama" {
		t.Errorf("events = %v", got)
	}
}

func TestAChatRejectionAndNoConnectionAreServedErrors(t *testing.T) {
	for name, want := range map[string]struct {
		err  error
		code v1.HostModelErrorCode
	}{
		"rejection":     {&connection.StoppedError{ConnectionName: "OpenRouter", Reason: "bad request"}, v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED},
		"no connection": {connection.ErrNoConnection, v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newModelClient(t, &scriptedCalls{err: want.err}, "token-a")
			stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
			if err != nil {
				t.Fatal(err)
			}
			got := readChat(t, stream)
			if len(got) != 1 || got[0].GetFailure().GetError() != want.code {
				t.Errorf("events = %v", got)
			}
		})
	}
}

func TestAHostWithNoModelCallsAnswersAChatWithNoConnection(t *testing.T) {
	client, _ := newModelClient(t, nil, "token-a")
	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)
	if len(got) != 1 || got[0].GetFailure().GetError() != v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("events = %v", got)
	}
}

func TestAnyOtherChatFailureIsAStatusCode(t *testing.T) {
	client, _ := newModelClient(t, &scriptedCalls{err: errors.New("the settings are unreadable")}, "token-a")
	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
	if err == nil {
		stream.Receive()
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("code = %v, want internal", connect.CodeOf(err))
	}
}

func TestACanceledChatReachesTheCalls(t *testing.T) {
	calls := &scriptedCalls{events: []modelcall.ChatEvent{{TextDelta: "partial"}}, started: make(chan struct{})}
	client, _ := newModelClient(t, calls, "token-a")
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	go func() {
		stream, err := client.Chat(ctx, connect.NewRequest(&v1.HostModelServiceChatRequest{}))
		if err == nil {
			for stream.Receive() {
			}
			err = stream.Err()
		}
		outcome <- err
	}()
	<-calls.started

	cancel()

	if err := <-outcome; connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("code = %v, want canceled: %v", connect.CodeOf(err), err)
	}
}

func TestAChatConsumerReceivesNoKeyInAnyEvent(t *testing.T) {
	const keyOne, keyTwo = "sk-one-secret-0123", "sk-two-secret-4567"
	for name, arrange := range map[string]func(*modelcalltest.World){
		"an answer": func(*modelcalltest.World) {},
		"every row failed": func(w *modelcalltest.World) {
			w.Provider.FailAlways("https://one.test/v1", &connection.TryNextError{Reason: connection.FailureReasonRejectedKey})
			w.Provider.FailAlways("https://two.test/v1", &connection.TryNextError{Reason: connection.FailureReasonPaymentRequired})
		},
		"a failure in the middle that echoes the key": func(w *modelcalltest.World) {
			w.Provider.ScriptChat("https://one.test/v1", errors.New("Bearer "+keyOne), modelcall.ChatEvent{TextDelta: "partial"})
		},
	} {
		world := modelcalltest.New()
		one := world.AddServer(t, "One", "https://one.test/v1", keyOne)
		two := world.AddServer(t, "Two", "https://two.test/v1", keyTwo)
		world.Rows.Set("md.numen.b", "chat", modelcalltest.Row(one, "m"), modelcalltest.Row(two, "m"))
		arrange(world)
		client, recording := newModelClient(t, modelcalls.NewCalls(lookupsOf(world), world.Source, world.Clock, world.Waits.Wait, world.Notices.ReportSetAside), "token-b")

		stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.HostModelServiceChatRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		for stream.Receive() {
		}

		if received := recording.received.String(); strings.Contains(received, "sk-") {
			t.Errorf("%s: the consumer received a key: %s", name, received)
		}
	}
}

func TestEveryChatFinishReasonOfTheSchemaIsEncodedFromThePortOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.HostModelChatFinishReason]modelcall.ChatFinishReason{
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP:       modelcall.ChatFinishStop,
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS: modelcall.ChatFinishToolCalls,
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_LENGTH:     modelcall.ChatFinishLength,
	}, writeChatFinishReason)
}
