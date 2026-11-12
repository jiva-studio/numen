package providerbridge_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// chatService is a ProviderService whose Chat sends the events the test sets, then waits for the context
// when block is set.
type chatService struct {
	numenv1connect.UnimplementedProviderServiceHandler
	events   []*v1.ProviderServiceChatResponse
	received *v1.ProviderServiceChatRequest
	block    bool
	started  chan struct{}
	ended    chan error
}

func (s *chatService) Chat(
	ctx context.Context, request *connect.Request[v1.ProviderServiceChatRequest], stream *connect.ServerStream[v1.ProviderServiceChatResponse],
) error {
	s.received = request.Msg
	for _, one := range s.events {
		if err := stream.Send(one); err != nil {
			return err
		}
	}
	if s.block {
		close(s.started)
		<-ctx.Done()
		s.ended <- ctx.Err()
		return ctx.Err()
	}
	return nil
}

func providerText(text string) *v1.ProviderServiceChatResponse {
	return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_TextDelta{TextDelta: text}}
}

func providerFinish(reason v1.ProviderChatFinishReason) *v1.ProviderServiceChatResponse {
	return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_Finish{Finish: &v1.ProviderChatFinish{Reason: reason}}}
}

func providerFailure(failure *v1.ProviderFailure) *v1.ProviderServiceChatResponse {
	return &v1.ProviderServiceChatResponse{Event: &v1.ProviderServiceChatResponse_Failure{Failure: failure}}
}

func collect(events *[]modelcall.ChatEvent) func(modelcall.ChatEvent) error {
	return func(event modelcall.ChatEvent) error {
		*events = append(*events, event)
		return nil
	}
}

func TestChatCarriesTheCallTheTurnAndStreamsTheEventsInOrder(t *testing.T) {
	service := &chatService{events: []*v1.ProviderServiceChatResponse{
		providerText("Let me "),
		providerText("look."),
		{Event: &v1.ProviderServiceChatResponse_ToolCall{ToolCall: &v1.ProviderToolCall{Id: "c1", Name: "list_notes", ArgumentsJson: `{"limit":3}`}}},
		providerFinish(v1.ProviderChatFinishReason_PROVIDER_CHAT_FINISH_REASON_TOOL_CALLS),
	}}
	provider, _, _ := newBridge(t, service)
	temperature := 0.5
	var got []modelcall.ChatEvent

	err := provider.Chat(t.Context(), modelcall.ProviderCall{Type: "server", Model: "m", Reasoning: "low", Secrets: map[string]string{"apiKey": "sk-1"}},
		modelcall.ChatRequest{
			Messages: []modelcall.ChatMessage{
				{Role: "user", Text: "list"},
				{Role: "assistant", ToolCalls: []modelcall.ToolCall{{ID: "c0", Name: "list_notes", Arguments: "{}"}}},
				{Role: "tool", Text: "a, b", ToolCallID: "c0"},
			},
			Tools:       []modelcall.ToolDefinition{{Name: "list_notes", Description: "Lists.", Parameters: `{"type":"object"}`}},
			Temperature: &temperature,
			MaxTokens:   10,
		}, collect(&got))

	if err != nil {
		t.Fatal(err)
	}
	finish := modelcall.ChatFinishToolCalls
	if len(got) != 4 || got[0].TextDelta != "Let me " || got[1].TextDelta != "look." ||
		*got[2].ToolCall != (modelcall.ToolCall{ID: "c1", Name: "list_notes", Arguments: `{"limit":3}`}) || got[3].Finish == nil || *got[3].Finish != finish {
		t.Errorf("events = %+v", got)
	}
	request := service.received
	if request.GetCall().GetModel() != "m" || request.GetCall().GetSecrets()[0].GetValue() != "sk-1" || request.GetTemperature() != 0.5 ||
		request.GetMaxTokens() != 10 || len(request.GetMessages()) != 3 || request.GetMessages()[1].GetToolCalls()[0].GetId() != "c0" ||
		request.GetMessages()[2].GetToolCallId() != "c0" || request.GetTools()[0].GetParametersJson() != `{"type":"object"}` {
		t.Errorf("request = %v", request)
	}
}

func TestAChatFailureBeforeTheFirstEventBecomesATryNextError(t *testing.T) {
	provider, _, _ := newBridge(t, &chatService{events: []*v1.ProviderServiceChatResponse{providerFailure(&v1.ProviderFailure{
		Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED, RetryAfterMs: 2500,
	})}})
	var got []modelcall.ChatEvent

	err := provider.Chat(t.Context(), modelcall.ProviderCall{}, modelcall.ChatRequest{}, collect(&got))

	var tryNext *connection.TryNextError
	if !errors.As(err, &tryNext) || tryNext.Reason != connection.FailureReasonRateLimited || tryNext.RetryAfter != 2500*time.Millisecond || len(got) != 0 {
		t.Errorf("err = %v, events = %v, want a try-next error for a rate limit after 2.5 s", err, got)
	}
}

func TestAChatFailureAfterTheFirstEventKeepsTheEventsAndBecomesAStopError(t *testing.T) {
	provider, _, _ := newBridge(t, &chatService{events: []*v1.ProviderServiceChatResponse{
		providerText("partial"),
		providerFailure(&v1.ProviderFailure{Kind: v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP, Reason: v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST}),
	}})
	var got []modelcall.ChatEvent

	err := provider.Chat(t.Context(), modelcall.ProviderCall{}, modelcall.ChatRequest{}, collect(&got))

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonInvalidRequest || len(got) != 1 || got[0].TextDelta != "partial" {
		t.Errorf("err = %v, events = %v", err, got)
	}
}

func TestAChatStreamThatEndsWithoutAFinishIsAnError(t *testing.T) {
	provider, _, _ := newBridge(t, &chatService{events: []*v1.ProviderServiceChatResponse{providerText("cut")}})
	var got []modelcall.ChatEvent

	err := provider.Chat(t.Context(), modelcall.ProviderCall{}, modelcall.ChatRequest{}, collect(&got))

	var tryNext *connection.TryNextError
	var stopError *connection.StopError
	if err == nil || errors.As(err, &tryNext) || errors.As(err, &stopError) {
		t.Errorf("err = %v, want a plain error the call path treats as unreachable", err)
	}
}

func TestAChatRoleThePluginDoesNotServeIsAStopOfUnknownReason(t *testing.T) {
	provider, _, _ := newBridge(t, numenv1connect.UnimplementedProviderServiceHandler{})

	err := provider.Chat(t.Context(), modelcall.ProviderCall{}, modelcall.ChatRequest{}, func(modelcall.ChatEvent) error { return nil })

	var stopError *connection.StopError
	if !errors.As(err, &stopError) || stopError.Reason != connection.FailureReasonUnavailable {
		t.Errorf("err = %v, want a stop error of unknown reason", err)
	}
}

func TestAnErrorOfTheReceiverEndsTheChatAndIsReturnedAsItIs(t *testing.T) {
	provider, _, _ := newBridge(t, &chatService{events: []*v1.ProviderServiceChatResponse{providerText("a"), providerText("b")}})
	consumerFailure := errors.New("the consumer went away")

	err := provider.Chat(t.Context(), modelcall.ProviderCall{}, modelcall.ChatRequest{}, func(modelcall.ChatEvent) error { return consumerFailure })

	if !errors.Is(err, consumerFailure) {
		t.Errorf("err = %v, want the error of the receiver", err)
	}
}

func cancelRunningChat(t *testing.T, provider connectionport.Provider, service *chatService) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	outcome := make(chan error, 1)
	var got []modelcall.ChatEvent
	go func() {
		outcome <- provider.Chat(ctx, modelcall.ProviderCall{}, modelcall.ChatRequest{}, collect(&got))
	}()
	select {
	case <-service.started:
	case <-time.After(cancelWait):
		t.Fatal("the provider never received the turn")
	}
	cancel()
	select {
	case got := <-service.ended:
		if !errors.Is(got, context.Canceled) {
			t.Fatalf("the handler context ended with %v, want context.Canceled", got)
		}
	case <-time.After(cancelWait):
		t.Fatal("the context of the provider's handler did not end after the caller canceled")
	}
	select {
	case err := <-outcome:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Chat = %v, want context.Canceled", err)
		}
	case <-time.After(cancelWait):
		t.Fatal("Chat did not return after it was canceled")
	}
}

func newBlockedChat() *chatService {
	return &chatService{
		events: []*v1.ProviderServiceChatResponse{providerText("partial")}, block: true,
		started: make(chan struct{}), ended: make(chan error, 1),
	}
}

func TestACanceledChatReachesTheProviderOverTheInProcessPipe(t *testing.T) {
	service := newBlockedChat()
	provider, _, _ := newBridge(t, service)
	cancelRunningChat(t, provider, service)
}

func TestACanceledChatReachesTheProviderOverTheLoopback(t *testing.T) {
	service := newBlockedChat()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewProviderServiceHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	address, isTCP := server.Listener.Addr().(*net.TCPAddr)
	if !isTCP {
		t.Fatal("the test server does not listen on TCP")
	}
	host := &activator{endpoint: transport.Endpoint{Host: address.IP.String(), Port: address.Port, Token: token}}
	provider, _, err := providerbridge.NewSource(host).OpenProvider(t.Context(), pluginID)
	if err != nil {
		t.Fatal(err)
	}
	cancelRunningChat(t, provider, service)
}
