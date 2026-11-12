package connection

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

type scriptedChatter struct {
	events   []ChatEvent
	err      error
	received Call
	request  ProviderChatRequest
	entered  chan struct{}
	ended    chan error
}

func (c *scriptedChatter) Chat(ctx context.Context, call Call, request ProviderChatRequest, send func(ChatEvent) error) error {
	c.received, c.request = call, request
	if c.entered != nil {
		close(c.entered)
		<-ctx.Done()
		c.ended <- ctx.Err()
		return ctx.Err()
	}
	for _, one := range c.events {
		if err := send(one); err != nil {
			return err
		}
	}
	return c.err
}

func newChatRequest() *connect.Request[v1.ProviderServiceChatRequest] {
	temperature := 0.25
	return connect.NewRequest(&v1.ProviderServiceChatRequest{
		Call: &v1.ProviderCall{
			Type: "openrouter", Model: "m", Reasoning: "low", Fields: map[string]string{"baseUrl": "https://x"},
			Secrets: []*v1.ProviderSecret{{Key: "apiKey", Value: "sk-1"}},
		},
		Messages: []*v1.ProviderChatMessage{
			{Role: "user", Text: "list the notes"},
			{Role: "assistant", ToolCalls: []*v1.ProviderToolCall{{Id: "call-1", Name: "list_notes", ArgumentsJson: `{"limit":3}`}}},
			{Role: "tool", Text: "a, b, c", ToolCallId: "call-1"},
		},
		Tools: []*v1.ProviderToolDefinition{{
			Name: "list_notes", Description: "Lists the notes.", ParametersJson: `{"type":"object","properties":{}}`,
		}},
		Temperature: &temperature,
		MaxTokens:   64,
	})
}

func readChatStream(t *testing.T, client interface {
	Chat(context.Context, *connect.Request[v1.ProviderServiceChatRequest]) (*connect.ServerStreamForClient[v1.ProviderServiceChatResponse], error)
}) []*v1.ProviderServiceChatResponse {
	t.Helper()
	stream, err := client.Chat(t.Context(), newChatRequest())
	if err != nil {
		t.Fatal(err)
	}
	var got []*v1.ProviderServiceChatResponse
	for stream.Receive() {
		got = append(got, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	return got
}

func TestAChatCallReachesTheProviderWithItsSnapshotMessagesAndTools(t *testing.T) {
	chatter := &scriptedChatter{events: []ChatEvent{{Finish: &Finish{}}}}
	client := newProviderClient(t, Provider{Chatter: chatter})

	readChatStream(t, client)

	if chatter.received.Type != "openrouter" || chatter.received.Model != "m" || chatter.received.Reasoning != "low" ||
		chatter.received.Secrets["apiKey"].Value != "sk-1" || chatter.received.Fields["baseUrl"] != "https://x" {
		t.Errorf("call = %#v", chatter.received)
	}
	request := chatter.request
	if request.MaxTokens != 64 || request.Temperature == nil || *request.Temperature != 0.25 || len(request.Messages) != 3 {
		t.Fatalf("request = %#v", request)
	}
	if request.Messages[1].ToolCalls[0] != (ToolCall{ID: "call-1", Name: "list_notes", Arguments: `{"limit":3}`}) ||
		request.Messages[2].ToolCallID != "call-1" || request.Messages[2].Role != "tool" {
		t.Errorf("messages = %#v", request.Messages)
	}
	if len(request.Tools) != 1 || request.Tools[0].Name != "list_notes" || string(request.Tools[0].Parameters) != `{"type":"object","properties":{}}` {
		t.Errorf("tools = %#v", request.Tools)
	}
}

func TestAChatCallStreamsTextThenToolCallsThenTheFinish(t *testing.T) {
	chatter := &scriptedChatter{events: []ChatEvent{
		{TextDelta: "Let me "},
		{TextDelta: "look."},
		{ToolCall: &ToolCall{ID: "call-2", Name: "read_note", Arguments: `{"path":"a.md"}`}},
		{Finish: &Finish{Reason: FinishToolCalls}},
	}}
	client := newProviderClient(t, Provider{Chatter: chatter})

	got := readChatStream(t, client)

	if len(got) != 4 || got[0].GetTextDelta() != "Let me " || got[1].GetTextDelta() != "look." ||
		got[2].GetToolCall().GetId() != "call-2" || got[2].GetToolCall().GetName() != "read_note" ||
		got[2].GetToolCall().GetArgumentsJson() != `{"path":"a.md"}` ||
		got[3].GetFinish().GetReason() != v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS {
		t.Errorf("events = %v", got)
	}
}

func TestAFailureBeforeTheFirstEventIsTheOnlyEventOfTheStream(t *testing.T) {
	client := newProviderClient(t, Provider{Chatter: &scriptedChatter{err: NewTryNextError(ReasonRateLimited, 7*time.Second)}})

	got := readChatStream(t, client)

	failure := got[0].GetFailure()
	if len(got) != 1 || failure.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT ||
		failure.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED || failure.GetRetryAfterMs() != 7000 {
		t.Errorf("events = %v, want one try-next failure", got)
	}
}

func TestAFailureAfterTheFirstEventEndsTheStream(t *testing.T) {
	chatter := &scriptedChatter{events: []ChatEvent{{TextDelta: "partial"}}, err: NewTryNextError(ReasonUnreachable, 0)}
	client := newProviderClient(t, Provider{Chatter: chatter})

	got := readChatStream(t, client)

	if len(got) != 2 || got[0].GetTextDelta() != "partial" ||
		got[1].GetFailure().GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE {
		t.Errorf("events = %v, want the text then the failure", got)
	}
}

func TestAChatErrorOfNoKindIsAStopWhoseTextDoesNotLeave(t *testing.T) {
	client := newProviderClient(t, Provider{Chatter: &scriptedChatter{err: errPlain}})

	got := readChatStream(t, client)

	failure := got[0].GetFailure()
	if failure.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP ||
		failure.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN {
		t.Errorf("failure = %v, want a stop of unknown reason", failure)
	}
}

func TestACancelledChatEndsTheContextOfTheProvider(t *testing.T) {
	chatter := &scriptedChatter{entered: make(chan struct{}), ended: make(chan error, 1)}
	client := newProviderClient(t, Provider{Chatter: chatter})
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		stream, err := client.Chat(ctx, newChatRequest())
		if err == nil {
			closeStreamAfter(t, stream)
		}
	}()
	<-chatter.entered

	cancel()

	select {
	case got := <-chatter.ended:
		if !errors.Is(got, context.Canceled) {
			t.Errorf("provider context error = %v, want canceled", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the context of the provider did not end after the client canceled")
	}
}

func TestChatIsUnimplementedWhereTheProviderHasNoChatter(t *testing.T) {
	client := newProviderClient(t, Provider{Completer: &scriptedCompleter{}})
	stream, err := client.Chat(t.Context(), newChatRequest())
	if err == nil {
		for stream.Receive() {
		}
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("code = %v, want unimplemented: %v", connect.CodeOf(err), err)
	}
}

func TestEveryFinishReasonOfTheSchemaIsEncoded(t *testing.T) {
	schematest.CheckProduced(t, map[v1.ProviderChatFinishReason]FinishReason{
		v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_STOP:       FinishStop,
		v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS: FinishToolCalls,
		v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_LENGTH:     FinishLength,
	}, writeFinishReason)
}

func TestAToolCallRoundTripsThroughTheWire(t *testing.T) {
	call := ToolCall{ID: "c", Name: "n", Arguments: `{"a":1}`}
	if got := readProviderToolCall(writeProviderToolCall(call)); got != call {
		t.Errorf("round trip = %#v, want %#v", got, call)
	}
}
