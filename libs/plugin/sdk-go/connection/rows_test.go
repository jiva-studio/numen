package connection

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
)

// rowsHost answers the listings with the row of the connection Meadow and records what it was asked.
type rowsHost struct {
	fakeHost
	listed  *v1.HostConnectionServiceListRowsRequest
	options *v1.HostConnectionServiceListRowOptionsRequest
}

func (h *rowsHost) ListRows(
	_ context.Context, request *connect.Request[v1.HostConnectionServiceListRowsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowsResponse], error) {
	h.token, h.listed = request.Header().Get("Authorization"), request.Msg
	return connect.NewResponse(&v1.HostConnectionServiceListRowsResponse{Rows: []*v1.CallRow{{
		ConnectionId: "c-meadow", ConnectionName: "Meadow", Options: map[string]string{"model": "wren-2"},
		Fields: []*v1.CallRowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models", IsRequired: true}},
	}}}), nil
}

func (h *rowsHost) ListRowOptions(
	_ context.Context, request *connect.Request[v1.HostConnectionServiceListRowOptionsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowOptionsResponse], error) {
	h.options = request.Msg
	if request.Msg.GetConnectionId() != "c-meadow" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("not among the rows"))
	}
	return connect.NewResponse(&v1.HostConnectionServiceListRowOptionsResponse{
		Options: []*v1.Option{{Value: "wren-2", Title: "Wren 2", Group: "Birds"}},
	}), nil
}

func newRowsClient(t *testing.T, host *rowsHost) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostConnectionServiceHandler(host))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return NewClient(server.Client(), server.URL, "token-1")
}

func TestListRowsReadsTheRowsOfTheOperation(t *testing.T) {
	host := &rowsHost{}

	rows, err := newRowsClient(t, host).ListRows(t.Context(), "llm.chat")

	want := []Row{{
		ConnectionID: "c-meadow", ConnectionName: "Meadow", Options: map[string]string{"model": "wren-2"},
		Fields: []RowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models", IsRequired: true}},
	}}
	if err != nil || !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows = %+v, %v, want %+v", rows, err, want)
	}
	if host.token != "Bearer token-1" || host.listed.GetOperation() != "llm.chat" {
		t.Errorf("token = %q, request = %v", host.token, host.listed)
	}
}

func TestListRowOptionsReadsTheValuesOfTheRowField(t *testing.T) {
	host := &rowsHost{}

	options, err := newRowsClient(t, host).ListRowOptions(t.Context(), RowOptionsRequest{
		Operation: "llm.chat", ConnectionID: "c-meadow", Field: "model", Options: map[string]string{"reasoning": "low"},
	})

	if err != nil || !reflect.DeepEqual(options, []Option{{Value: "wren-2", Title: "Wren 2", Group: "Birds"}}) {
		t.Fatalf("options = %+v, %v", options, err)
	}
	asked := host.options
	if asked.GetOperation() != "llm.chat" || asked.GetField() != "model" || asked.GetOptions()["reasoning"] != "low" {
		t.Errorf("request = %v", asked)
	}
}

func TestListRowOptionsOfAConnectionThatIsNotAmongTheRowsFails(t *testing.T) {
	_, err := newRowsClient(t, &rowsHost{}).ListRowOptions(t.Context(), RowOptionsRequest{
		Operation: "llm.chat", ConnectionID: "c-elsewhere", Field: "model", Options: nil,
	})

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("err = %v, want not found", err)
	}
}

func TestACallOfAPreferringCallerNamesThePreferredRow(t *testing.T) {
	host := &rowsHost{fakeHost: fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostDone("finish")}}}
	client := newRowsClient(t, host)

	frames, err := client.Prefer(PreferredRow{ConnectionID: "c-meadow", Options: map[string]string{"model": "wren-2"}}).
		Call(t.Context(), "llm.chat", []byte("list the notes"), PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if _, done, err := Collect(frames); err != nil || string(done) != "finish" {
		t.Fatalf("done = %q, %v", done, err)
	}

	preferred := host.received.GetPreferred()
	if preferred.GetConnectionId() != "c-meadow" || preferred.GetOptions()["model"] != "wren-2" ||
		host.received.GetOperation() != "llm.chat" || host.received.GetPriority() != v1.Priority_PRIORITY_INTERACTIVE {
		t.Errorf("request = %v", host.received)
	}
}

func TestACallOfAPreferringCallerNamesAnExclusiveRow(t *testing.T) {
	host := &rowsHost{fakeHost: fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostDone("finish")}}}

	frames, err := newRowsClient(t, host).Prefer(PreferredRow{ConnectionID: "c-meadow", IsExclusive: true}).
		Call(t.Context(), "llm.chat", nil, PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Collect(frames); err != nil {
		t.Fatal(err)
	}

	if !host.received.GetPreferred().GetIsExclusive() {
		t.Errorf("preferred = %v, want an exclusive row", host.received.GetPreferred())
	}
}

func TestACallOfTheClientNamesNoPreferredRow(t *testing.T) {
	host := &rowsHost{fakeHost: fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostDone("finish")}}}

	frames, err := newRowsClient(t, host).Call(t.Context(), "llm.chat", nil, PriorityBackground)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Collect(frames); err != nil {
		t.Fatal(err)
	}

	if host.received.GetPreferred() != nil {
		t.Errorf("preferred = %v, want none", host.received.GetPreferred())
	}
}
