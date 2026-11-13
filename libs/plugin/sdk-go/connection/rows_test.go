package connection

import (
	"context"
	"errors"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var errNotAmongTheRows = errors.New("the connection is not among the rows")

func (h *fakeHost) ListRows(
	_ context.Context, request *connect.Request[v1.HostConnectionServiceListRowsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowsResponse], error) {
	h.token = request.Header().Get("Authorization")
	if request.Msg.GetOperation() != "acp.task" {
		return connect.NewResponse(&v1.HostConnectionServiceListRowsResponse{}), nil
	}
	return connect.NewResponse(&v1.HostConnectionServiceListRowsResponse{Rows: []*v1.CallRow{{
		ConnectionId: "c-claude", ConnectionName: "Claude", Options: map[string]string{"model": "opus"},
		Fields: []*v1.CallRowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models"}},
	}}}), nil
}

func (h *fakeHost) ListRowOptions(
	_ context.Context, request *connect.Request[v1.HostConnectionServiceListRowOptionsRequest],
) (*connect.Response[v1.HostConnectionServiceListRowOptionsResponse], error) {
	message := request.Msg
	if message.GetConnectionId() != "c-claude" || message.GetField() != "reasoning" || message.GetOptions()["model"] != "opus" {
		return nil, connect.NewError(connect.CodeNotFound, errNotAmongTheRows)
	}
	return connect.NewResponse(&v1.HostConnectionServiceListRowOptionsResponse{Options: []*v1.Option{{Value: "high", Title: "High"}}}), nil
}

func TestACallCarriesTheRowItPrefers(t *testing.T) {
	host := &fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostDone("")}}
	ctx := WithPreferredRow(t.Context(), PreferredRow{ConnectionID: "c-claude", Options: map[string]string{"model": "opus"}})

	frames, err := newTestClient(t, host).Call(ctx, "acp.task", nil, PriorityInteractive)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Collect(frames); err != nil {
		t.Fatal(err)
	}

	preferred := host.received.GetPreferred()
	if preferred.GetConnectionId() != "c-claude" || preferred.GetOptions()["model"] != "opus" {
		t.Errorf("preferred = %v", preferred)
	}
}

func TestACallWithoutAPreferredRowCarriesNone(t *testing.T) {
	host := &fakeHost{frames: []*v1.HostConnectionServiceCallResponse{hostDone("")}}

	if _, _, err := collectCall(t, newTestClient(t, host), PriorityBackground); err != nil {
		t.Fatal(err)
	}

	if host.received.GetPreferred() != nil {
		t.Errorf("preferred = %v, want none", host.received.GetPreferred())
	}
}

func TestTheRowsOfAnOperationAreListedWithTheirConnectionAndFields(t *testing.T) {
	host := &fakeHost{}

	rows, err := newTestClient(t, host).ListRows(t.Context(), "acp.task")

	want := []RowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models"}}
	if err != nil || len(rows) != 1 || rows[0].ConnectionID != "c-claude" || rows[0].ConnectionName != "Claude" ||
		rows[0].Options["model"] != "opus" || !slices.Equal(rows[0].Fields, want) {
		t.Fatalf("ListRows = %+v, %v", rows, err)
	}
	if host.token != "Bearer token-1" {
		t.Errorf("token = %q", host.token)
	}
}

func TestTheValuesOfARowFieldAreListed(t *testing.T) {
	listed, err := newTestClient(t, &fakeHost{}).ListRowOptions(t.Context(), "acp.task", "c-claude", "reasoning", map[string]string{"model": "opus"})

	if err != nil || !slices.Equal(listed, []Option{{Value: "high", Title: "High"}}) {
		t.Fatalf("ListRowOptions = %+v, %v", listed, err)
	}
}

func TestTheValuesOfARowFieldOfAConnectionOutsideTheRowsAreNotFound(t *testing.T) {
	_, err := newTestClient(t, &fakeHost{}).ListRowOptions(t.Context(), "acp.task", "c-other", "model", nil)

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("err = %v, want not found", err)
	}
}
