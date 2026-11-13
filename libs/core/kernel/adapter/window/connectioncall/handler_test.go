package connectioncall_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// scriptedCalls sends the frames the test sets and answers done or err, recording the plugin and request.
type scriptedCalls struct {
	frames  []string
	done    string
	err     error
	plugin  string
	request modelcall.Request
}

func (s *scriptedCalls) Call(_ context.Context, pluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error) {
	s.plugin, s.request = pluginID, request
	for _, one := range s.frames {
		if err := send([]byte(one)); err != nil {
			return nil, err
		}
	}
	if s.err != nil {
		return nil, s.err
	}
	return []byte(s.done), nil
}

func newClient(t *testing.T, calls *scriptedCalls, options ...connect.ClientOption) numenv1connect.ConnectionCallServiceClient {
	t.Helper()
	service := connectioncall.NewService(nil, nil)
	if calls != nil {
		service = connectioncall.NewService(calls, calls)
	}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewConnectionCallServiceHandler(service))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewConnectionCallServiceClient(server.Client(), server.URL, options...)
}

type received struct {
	frames  []string
	done    string
	failure *v1.CallFailure
}

func call(t *testing.T, client numenv1connect.ConnectionCallServiceClient, request *connect.Request[v1.ConnectionCallServiceCallRequest]) (received, error) {
	t.Helper()
	stream, err := client.Call(t.Context(), request)
	if err != nil {
		return received{}, err
	}
	var got received
	for stream.Receive() {
		switch frame := stream.Msg().GetFrame().(type) {
		case *v1.ConnectionCallServiceCallResponse_Data:
			got.frames = append(got.frames, string(frame.Data))
		case *v1.ConnectionCallServiceCallResponse_Done:
			got.done = string(frame.Done)
		case *v1.ConnectionCallServiceCallResponse_Failure:
			got.failure = frame.Failure
		}
	}
	return got, stream.Err()
}

func TestACallStreamsTheFramesThroughTheRowsOfThePluginTheRequestNames(t *testing.T) {
	calls := &scriptedCalls{frames: []string{"Front: ", "the capital"}, done: "finish"}
	client := newClient(t, calls)

	got, err := call(t, client, connect.NewRequest(&v1.ConnectionCallServiceCallRequest{
		PluginId: "md.numen.views.flashcards", Operation: "llm.chat", Payload: []byte("make a card"), Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))

	if err != nil || !slices.Equal(got.frames, []string{"Front: ", "the capital"}) || got.done != "finish" || got.failure != nil {
		t.Fatalf("Call = %+v, %v", got, err)
	}
	if calls.plugin != "md.numen.views.flashcards" || calls.request.Operation != "llm.chat" ||
		string(calls.request.Payload) != "make a card" || calls.request.Priority != modelcall.PriorityInteractive {
		t.Errorf("plugin = %q, request = %+v", calls.plugin, calls.request)
	}
}

func TestACallTheCoreCouldNotServeEndsWithAFailureFrame(t *testing.T) {
	for name, want := range map[string]struct {
		err  error
		code v1.CallErrorCode
	}{
		"no row":    {connection.ErrNoConnection, v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION},
		"every row": {&connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: "Ollama", Reason: "unavailable"}}}, v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED},
		"a stop":    {&connection.StoppedError{ConnectionName: "OpenRouter", Reason: "invalid request"}, v1.CallErrorCode_CALL_ERROR_CODE_STOPPED},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := call(t, newClient(t, &scriptedCalls{err: want.err}), connect.NewRequest(&v1.ConnectionCallServiceCallRequest{Operation: "llm.complete"}))

			if err != nil || got.failure.GetError() != want.code {
				t.Errorf("Call = %+v, %v, want %v", got, err, want.code)
			}
		})
	}
}

func TestACallThatCarriesAPluginTokenIsRefused(t *testing.T) {
	calls := &scriptedCalls{done: "finish"}
	client := newClient(t, calls)
	request := connect.NewRequest(&v1.ConnectionCallServiceCallRequest{PluginId: "md.numen.views.flashcards", Operation: "llm.complete"})
	request.Header().Set("Authorization", "Bearer plugin-token")

	_, err := call(t, client, request)

	if connect.CodeOf(err) != connect.CodePermissionDenied || calls.plugin != "" {
		t.Errorf("code = %v, plugin = %q, want permission denied and no call", connect.CodeOf(err), calls.plugin)
	}
}

func TestAnInstallationWithNoCallsAnswersUnimplemented(t *testing.T) {
	_, err := call(t, newClient(t, nil), connect.NewRequest(&v1.ConnectionCallServiceCallRequest{Operation: "llm.complete"}))

	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("code = %v, want unimplemented", connect.CodeOf(err))
	}
}

func (s *scriptedCalls) ListRows(pluginID, operation string) ([]modelcall.Row, error) {
	s.plugin, s.request = pluginID, modelcall.Request{Operation: operation}
	return []modelcall.Row{{
		ConnectionID: "c-claude", ConnectionName: "Claude", Options: map[string]string{"model": "opus"},
		Fields: []modelcall.RowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models"}},
	}}, nil
}

func (s *scriptedCalls) ListRowOptions(
	_ context.Context, pluginID, operation, connectionID, field string, options map[string]string,
) ([]modelcall.Option, error) {
	s.plugin, s.request = pluginID, modelcall.Request{Operation: operation}
	if connectionID != "c-claude" || field != "reasoning" || options["model"] != "opus" {
		return nil, connection.ErrNotFound
	}
	return []modelcall.Option{{Value: "high", Title: "High"}}, nil
}

func TestACallCarriesThePreferredRowOfTheRequest(t *testing.T) {
	calls := &scriptedCalls{done: "finish"}
	client := newClient(t, calls)

	_, err := call(t, client, connect.NewRequest(&v1.ConnectionCallServiceCallRequest{
		PluginId: "md.numen.views.agent.chat", Operation: "acp.task",
		Preferred: &v1.PreferredRow{ConnectionId: "c-claude", Options: map[string]string{"model": "opus"}},
	}))

	preferred := calls.request.Preferred
	if err != nil || preferred == nil || preferred.ConnectionID != "c-claude" || preferred.Options["model"] != "opus" {
		t.Fatalf("preferred = %+v, %v", preferred, err)
	}
}

func TestACallWithoutAPreferredRowCarriesNone(t *testing.T) {
	calls := &scriptedCalls{done: "finish"}
	client := newClient(t, calls)

	_, err := call(t, client, connect.NewRequest(&v1.ConnectionCallServiceCallRequest{PluginId: "md.numen.views.agent.chat", Operation: "acp.task"}))

	if err != nil || calls.request.Preferred != nil {
		t.Fatalf("preferred = %+v, %v, want none", calls.request.Preferred, err)
	}
}

func TestTheRowsOfThePluginTheRequestNamesAreListed(t *testing.T) {
	calls := &scriptedCalls{}
	client := newClient(t, calls)

	response, err := client.ListRows(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceListRowsRequest{
		PluginId: "md.numen.views.agent.chat", Operation: "acp.task",
	}))
	if err != nil {
		t.Fatal(err)
	}

	rows := response.Msg.GetRows()
	if calls.plugin != "md.numen.views.agent.chat" || calls.request.Operation != "acp.task" || len(rows) != 1 {
		t.Fatalf("plugin = %q, operation = %q, rows = %v", calls.plugin, calls.request.Operation, rows)
	}
	row := rows[0]
	if row.GetConnectionId() != "c-claude" || row.GetConnectionName() != "Claude" || row.GetOptions()["model"] != "opus" ||
		len(row.GetFields()) != 1 || row.GetFields()[0].GetOptionsFrom() != "llm.models" {
		t.Errorf("row = %v", row)
	}
}

func TestTheValuesOfARowFieldAreListed(t *testing.T) {
	client := newClient(t, &scriptedCalls{})

	response, err := client.ListRowOptions(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceListRowOptionsRequest{
		PluginId: "md.numen.views.agent.chat", Operation: "acp.task", ConnectionId: "c-claude", Field: "reasoning",
		Options: map[string]string{"model": "opus"},
	}))

	if err != nil || len(response.Msg.GetOptions()) != 1 || response.Msg.GetOptions()[0].GetTitle() != "High" {
		t.Fatalf("ListRowOptions = %v, %v", response, err)
	}
}

func TestTheValuesOfARowFieldOfAConnectionOutsideTheRowsAreNotFound(t *testing.T) {
	client := newClient(t, &scriptedCalls{})

	_, err := client.ListRowOptions(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceListRowOptionsRequest{
		PluginId: "md.numen.views.agent.chat", Operation: "acp.task", ConnectionId: "c-other", Field: "model",
	}))

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestAListingThatCarriesAPluginTokenIsRefused(t *testing.T) {
	client := newClient(t, &scriptedCalls{})
	request := connect.NewRequest(&v1.ConnectionCallServiceListRowsRequest{PluginId: "md.numen.views.agent.chat", Operation: "acp.task"})
	request.Header().Set("Authorization", "Bearer token")

	_, err := client.ListRows(t.Context(), request)

	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("err = %v, want permission denied", err)
	}
}
