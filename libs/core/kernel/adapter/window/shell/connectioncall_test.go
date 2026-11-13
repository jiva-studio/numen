package shell

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/connectioncall"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

type recordedCall struct {
	pluginID string
	request  modelcall.Request
}

type answeringCalls struct{ calls chan recordedCall }

func (a *answeringCalls) Call(_ context.Context, pluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error) {
	a.calls <- recordedCall{pluginID: pluginID, request: request}
	if err := send([]byte("a frame")); err != nil {
		return nil, err
	}
	return []byte("done"), nil
}

func callConnection(t *testing.T, api *API) []*v1.ConnectionCallServiceCallResponse {
	t.Helper()
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewConnectionCallServiceClient(server.Client(), server.URL)
	stream, err := client.Call(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceCallRequest{
		PluginId: "org.example.cards", Operation: "acp.task", Payload: []byte("Where does it live?"),
		Priority: v1.Priority_PRIORITY_INTERACTIVE,
	}))
	if err != nil {
		t.Fatal(err)
	}
	var frames []*v1.ConnectionCallServiceCallResponse
	for stream.Receive() {
		frames = append(frames, stream.Msg())
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	return frames
}

func TestTheWindowCarriesOutTheConnectionCallsOfItsPlugins(t *testing.T) {
	calls := &answeringCalls{calls: make(chan recordedCall, 1)}
	api := newAPI(t)
	api.ConnectionCalls = connectioncall.NewHandler(calls)

	frames := callConnection(t, api)

	if len(frames) != 2 || string(frames[0].GetData()) != "a frame" || string(frames[1].GetDone()) != "done" {
		t.Fatalf("the frames are %v", frames)
	}
	got := <-calls.calls
	if got.pluginID != "org.example.cards" || got.request.Operation != "acp.task" ||
		string(got.request.Payload) != "Where does it live?" || got.request.Priority != modelcall.PriorityInteractive {
		t.Errorf("the call is %+v", got)
	}
}

func TestAWindowWithoutConnectionCallsAnswersThemAsUnimplemented(t *testing.T) {
	server := httptest.NewServer(newAPI(t).NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewConnectionCallServiceClient(server.Client(), server.URL)

	stream, err := client.Call(t.Context(), connect.NewRequest(&v1.ConnectionCallServiceCallRequest{PluginId: "org.example.cards"}))
	if err == nil {
		for stream.Receive() {
			t.Errorf("a frame arrived: %v", stream.Msg())
		}
		err = stream.Err()
	}
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Errorf("the error is %v, want unimplemented", err)
	}
}

func (*answeringCalls) ListRows(string, string) ([]modelcall.Row, error) { return nil, nil }

func (*answeringCalls) ListRowOptions(context.Context, string, string, string, string, map[string]string) ([]modelcall.Option, error) {
	return nil, nil
}
