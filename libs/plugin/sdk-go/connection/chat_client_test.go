package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

type fakeChatHost struct {
	numenv1connect.UnimplementedHostModelServiceHandler
	token   string
	events  []*v1.HostModelServiceChatResponse
	request *v1.HostModelServiceChatRequest
	entered chan struct{}
	ended   chan error
}

func (h *fakeChatHost) Chat(
	ctx context.Context, request *connect.Request[v1.HostModelServiceChatRequest], stream *connect.ServerStream[v1.HostModelServiceChatResponse],
) error {
	h.token, h.request = request.Header().Get("Authorization"), request.Msg
	for _, one := range h.events {
		if err := stream.Send(one); err != nil {
			return err
		}
	}
	if h.entered != nil {
		close(h.entered)
		<-ctx.Done()
		h.ended <- ctx.Err()
		return ctx.Err()
	}
	return nil
}

func newChatClient(t *testing.T, host *fakeChatHost) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostModelServiceHandler(host))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.Client(), server.URL, "token-1")
}

func textDelta(text string) *v1.HostModelServiceChatResponse {
	return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_TextDelta{TextDelta: text}}
}

func chatFinish(reason v1.HostModelChatFinishReason) *v1.HostModelServiceChatResponse {
	return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_Finish{Finish: &v1.HostModelChatFinish{Reason: reason}}}
}

func chatFailure(failure *v1.HostModelChatError) *v1.HostModelServiceChatResponse {
	return &v1.HostModelServiceChatResponse{Event: &v1.HostModelServiceChatResponse_Failure{Failure: failure}}
}

func readEvents(stream *ChatStream) []ChatEvent {
	var got []ChatEvent
	for {
		event, ok := stream.Next()
		if !ok {
			return got
		}
		got = append(got, event)
	}
}

func TestChatSendsTheTurnAndTheTokenAndStreamsTheEvents(t *testing.T) {
	host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{
		textDelta("Let me "),
		textDelta("look."),
		{Event: &v1.HostModelServiceChatResponse_ToolCall{ToolCall: &v1.HostModelToolCall{Id: "c1", Name: "list_notes", ArgumentsJson: `{"limit":3}`}}},
		chatFinish(v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS),
	}}
	temperature := 0.5

	stream, err := newChatClient(t, host).Chat(t.Context(), ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Text: "list"},
			{Role: "assistant", ToolCalls: []ToolCall{{ID: "c0", Name: "list_notes", Arguments: "{}"}}},
			{Role: "tool", Text: "a, b", ToolCallID: "c0"},
		},
		Tools:       []ToolDefinition{{Name: "list_notes", Description: "Lists.", Parameters: []byte(`{"type":"object"}`)}},
		Temperature: &temperature,
		MaxTokens:   100,
		Priority:    PriorityInteractive,
	})
	if err != nil {
		t.Fatal(err)
	}
	closeStreamAfter(t, stream)
	got := readEvents(stream)

	if stream.Err() != nil {
		t.Fatalf("Err = %v", stream.Err())
	}
	if len(got) != 4 || got[0].TextDelta != "Let me " || got[1].TextDelta != "look." ||
		got[2].ToolCall == nil || *got[2].ToolCall != (ToolCall{ID: "c1", Name: "list_notes", Arguments: `{"limit":3}`}) ||
		got[3].Finish == nil || got[3].Finish.Reason != FinishToolCalls {
		t.Errorf("events = %#v", got)
	}
	if host.token != "Bearer token-1" {
		t.Errorf("Authorization = %q", host.token)
	}
	request := host.request
	if request.GetPriority() != v1.Priority_PRIORITY_INTERACTIVE || request.GetMaxTokens() != 100 || request.GetTemperature() != 0.5 ||
		len(request.GetMessages()) != 3 || request.GetMessages()[1].GetToolCalls()[0].GetId() != "c0" ||
		request.GetMessages()[2].GetToolCallId() != "c0" ||
		len(request.GetTools()) != 1 || request.GetTools()[0].GetParametersJson() != `{"type":"object"}` {
		t.Errorf("request = %v", request)
	}
}

func TestAChatThatEndsWithoutAFinishIsAnError(t *testing.T) {
	host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{textDelta("cut")}}
	stream, err := newChatClient(t, host).Chat(t.Context(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	closeStreamAfter(t, stream)

	got := readEvents(stream)

	if len(got) != 1 || !errors.Is(stream.Err(), ErrStreamEnded) {
		t.Errorf("events = %v, Err = %v, want the text and ErrStreamEnded", got, stream.Err())
	}
}

func TestAFailureBeforeTheFirstEventIsTheErrorOfTheStream(t *testing.T) {
	host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{
		chatFailure(&v1.HostModelChatError{
			Error: v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED,
			Failures: []*v1.HostModelFailure{
				{ConnectionName: "OpenRouter", Reason: "payment required"}, {ConnectionName: "Ollama", Reason: "unreachable"},
			},
		}),
	}}
	stream, err := newChatClient(t, host).Chat(t.Context(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	closeStreamAfter(t, stream)

	got := readEvents(stream)

	var failed *AllFailedError
	if len(got) != 0 || !errors.As(stream.Err(), &failed) || len(failed.Failures) != 2 ||
		failed.Failures[0] != (Failure{ConnectionName: "OpenRouter", Reason: "payment required"}) {
		t.Errorf("events = %v, Err = %v", got, stream.Err())
	}
}

func TestAFailureAfterTheFirstEventKeepsTheTextAndNamesTheConnection(t *testing.T) {
	host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{
		textDelta("partial"),
		chatFailure(&v1.HostModelChatError{
			Error:    v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_ALL_FAILED,
			Failures: []*v1.HostModelFailure{{ConnectionName: "Ollama", Reason: "unreachable"}},
		}),
	}}
	stream, err := newChatClient(t, host).Chat(t.Context(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	closeStreamAfter(t, stream)

	got := readEvents(stream)

	var failed *AllFailedError
	if len(got) != 1 || got[0].TextDelta != "partial" || !errors.As(stream.Err(), &failed) ||
		failed.Failures[0] != (Failure{ConnectionName: "Ollama", Reason: "unreachable"}) {
		t.Errorf("events = %v, Err = %v", got, stream.Err())
	}
}

func TestChatReportsNoConnectionAndRejection(t *testing.T) {
	for name, want := range map[string]struct {
		code v1.HostModelErrorCode
		is   func(error) bool
	}{
		"no connection": {v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_NO_CONNECTION, func(err error) bool { return errors.Is(err, ErrNoConnection) }},
		"rejected": {v1.HostModelErrorCode_HOST_MODEL_ERROR_CODE_REJECTED, func(err error) bool {
			var rejected *RejectedError
			return errors.As(err, &rejected) && rejected.ConnectionName == "OpenRouter"
		}},
	} {
		t.Run(name, func(t *testing.T) {
			host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{chatFailure(&v1.HostModelChatError{
				Error: want.code, Failures: []*v1.HostModelFailure{{ConnectionName: "OpenRouter", Reason: "bad request"}},
			})}}
			stream, err := newChatClient(t, host).Chat(t.Context(), ChatRequest{})
			if err != nil {
				t.Fatal(err)
			}
			closeStreamAfter(t, stream)
			readEvents(stream)
			if !want.is(stream.Err()) {
				t.Errorf("Err = %v", stream.Err())
			}
		})
	}
}

func TestACanceledChatInTheMiddleOfATurnKeepsTheTextAndReturnsTheContextError(t *testing.T) {
	host := &fakeChatHost{events: []*v1.HostModelServiceChatResponse{textDelta("partial")}, entered: make(chan struct{}), ended: make(chan error, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	stream, err := newChatClient(t, host).Chat(ctx, ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	closeStreamAfter(t, stream)
	first, ok := stream.Next()
	<-host.entered

	cancel()
	_, isMore := stream.Next()

	if !ok || first.TextDelta != "partial" || isMore || !errors.Is(stream.Err(), context.Canceled) {
		t.Errorf("first = %v %v, more = %v, Err = %v, want the text then context.Canceled", first, ok, isMore, stream.Err())
	}
	if got := <-host.ended; !errors.Is(got, context.Canceled) {
		t.Errorf("the context of the host ended with %v, want canceled", got)
	}
}

func TestACanceledChatBeforeTheFirstEventReturnsTheContextError(t *testing.T) {
	host := &fakeChatHost{entered: make(chan struct{}), ended: make(chan error, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	openResult := make(chan error, 1)
	go func() {
		stream, err := newChatClient(t, host).Chat(ctx, ChatRequest{})
		if stream != nil {
			closeStreamAfter(t, stream)
		}
		openResult <- err
	}()
	<-host.entered

	cancel()

	if err := <-openResult; !errors.Is(err, context.Canceled) {
		t.Errorf("Chat = %v, want context.Canceled", err)
	}
	if got := <-host.ended; !errors.Is(got, context.Canceled) {
		t.Errorf("the context of the host ended with %v, want canceled", got)
	}
}

func TestEveryFinishReasonOfTheHostSchemaIsRead(t *testing.T) {
	want := map[v1.HostModelChatFinishReason]FinishReason{
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP:       FinishStop,
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_TOOL_CALLS: FinishToolCalls,
		v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_LENGTH:     FinishLength,
	}
	schematest.CheckHandled(t, func(reason v1.HostModelChatFinishReason) bool {
		expected, isNamed := want[reason]
		return isNamed && readHostFinishReason(reason) == expected
	})
}

func closeStreamAfter(t *testing.T, stream interface{ Close() error }) {
	t.Helper()
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	})
}
