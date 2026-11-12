package model

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/modelcalltest"
)

func newChatClient(t *testing.T) (numenv1connect.ModelServiceClient, *modelcalltest.World) {
	t.Helper()
	handler, world, entry := newHandler(t)
	world.Rows.Set(viewPluginID, "chat", modelcalltest.Row(entry, "model-1"))
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewModelServiceHandler(handler))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewModelServiceClient(server.Client(), server.URL), world
}

func readChat(t *testing.T, stream *connect.ServerStreamForClient[v1.ModelServiceChatResponse]) []*v1.ModelServiceChatResponse {
	t.Helper()
	var got []*v1.ModelServiceChatResponse
	for stream.Receive() {
		got = append(got, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream: %v", err)
	}
	return got
}

func TestChatStreamsAnswerThroughTheRowsOfThePluginTheRequestNames(t *testing.T) {
	client, world := newChatClient(t)

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.ModelServiceChatRequest{
		PluginId: viewPluginID,
		Messages: []*v1.HostModelChatMessage{{Role: "user", Text: "list"}},
		Tools:    []*v1.HostModelToolDefinition{{Name: "list_notes", ParametersJson: `{"type":"object"}`}},
		Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 2 || got[0].GetTextDelta() != "answer from "+address ||
		got[1].GetFinish().GetReason() != v1.HostModelChatFinishReason_HOST_MODEL_CHAT_FINISH_REASON_STOP {
		t.Fatalf("events = %v", got)
	}
	request := world.Provider.Calls()[0].Chat
	if len(request.Messages) != 1 || request.Messages[0].Text != "list" || request.Tools[0].Name != "list_notes" {
		t.Errorf("request = %+v", request)
	}
}

func TestAChatForAPluginWithNoRowIsAServedNoConnection(t *testing.T) {
	client, _ := newChatClient(t)

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.ModelServiceChatRequest{PluginId: "md.example.stranger"}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 1 || got[0].GetFailure().GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_NO_CONNECTION {
		t.Errorf("events = %v", got)
	}
}

func TestAChatFailureAfterTheFirstEventKeepsTheTextAndNamesTheConnection(t *testing.T) {
	client, world := newChatClient(t)
	world.Provider.ScriptChat(address, &connection.TryNextError{Reason: connection.FailureReasonUnreachable},
		modelcall.ChatEvent{TextDelta: "partial"})

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.ModelServiceChatRequest{PluginId: viewPluginID}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	failures := got[len(got)-1].GetFailure().GetFailures()
	if len(got) != 2 || got[0].GetTextDelta() != "partial" ||
		got[1].GetFailure().GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_ALL_FAILED ||
		len(failures) != 1 || failures[0].GetConnectionName() != "First" || failures[0].GetReason() != "unreachable" {
		t.Errorf("events = %v", got)
	}
}

func TestARejectedChatIsAServedRejectedThatNamesTheConnection(t *testing.T) {
	client, world := newChatClient(t)
	world.Provider.FailAlways(address, &connection.StopError{Reason: connection.FailureReasonBadRequest})

	stream, err := client.Chat(t.Context(), connect.NewRequest(&v1.ModelServiceChatRequest{PluginId: viewPluginID}))
	if err != nil {
		t.Fatal(err)
	}
	got := readChat(t, stream)

	if len(got) != 1 || got[0].GetFailure().GetError() != v1.ModelErrorCode_MODEL_ERROR_CODE_REJECTED ||
		got[0].GetFailure().GetFailures()[0].GetConnectionName() != "First" {
		t.Errorf("events = %v", got)
	}
}

func TestAChatWithTheTokenOfAPluginIsRefused(t *testing.T) {
	client, world := newChatClient(t)
	request := connect.NewRequest(&v1.ModelServiceChatRequest{PluginId: viewPluginID})
	request.Header().Set("Authorization", "Bearer token-of-a-plugin")

	stream, err := client.Chat(t.Context(), request)
	if err == nil {
		stream.Receive()
		err = stream.Err()
	}

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("code = %v, want permission denied", connect.CodeOf(err))
	}
	if got := world.Provider.Calls(); len(got) != 0 {
		t.Errorf("the provider received %d calls for a refused request", len(got))
	}
}
