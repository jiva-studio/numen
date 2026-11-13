package hostendpoint

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// meadowRows are the rows of llm.chat of md.numen.b: the connection Meadow with its model field.
type meadowRows struct {
	plugin string
	asked  []string
}

func (r *meadowRows) ListRows(pluginID, operation string) ([]modelcall.Row, error) {
	r.plugin = pluginID
	r.asked = append(r.asked, operation)
	if pluginID != "md.numen.b" || operation != "llm.chat" {
		return []modelcall.Row{}, nil
	}
	return []modelcall.Row{{
		ConnectionID: "c-meadow", ConnectionName: "Meadow", Options: map[string]string{},
		Fields: []modelcall.RowField{{Key: "model", Title: "Model", OptionsFrom: "llm.models", IsRequired: true}},
	}}, nil
}

func (r *meadowRows) ListRowOptions(
	_ context.Context, pluginID, operation, connectionID, field string, _ map[string]string,
) ([]modelcall.Option, error) {
	r.plugin = pluginID
	if operation != "llm.chat" || connectionID != "c-meadow" || field != "model" {
		return nil, connection.ErrNotFound
	}
	return []modelcall.Option{{Value: "wren-2", Title: "Wren 2", Group: "Birds"}}, nil
}

func newRowsClient(t *testing.T, rows RowLists, token string) numenv1connect.HostConnectionServiceClient {
	t.Helper()
	fakeCallers := newCallers()
	fakeCallers.rowLists = rows
	client, _ := newConnectionClientFor(t, fakeCallers, token)
	return client
}

func TestListRowsListsTheRowsOfThePluginTheTokenNames(t *testing.T) {
	rows := &meadowRows{}
	client := newRowsClient(t, rows, "token-b")

	response, err := client.ListRows(t.Context(), connect.NewRequest(&v1.HostConnectionServiceListRowsRequest{Operation: "llm.chat"}))

	if err != nil {
		t.Fatal(err)
	}
	listed := response.Msg.GetRows()
	if rows.plugin != "md.numen.b" || len(listed) != 1 || listed[0].GetConnectionName() != "Meadow" {
		t.Fatalf("plugin = %q, rows = %v", rows.plugin, listed)
	}
	field := listed[0].GetFields()[0]
	if field.GetKey() != "model" || field.GetOptionsFrom() != "llm.models" || !field.GetIsRequired() {
		t.Errorf("field = %v", field)
	}
}

func TestListRowOptionsListsTheValuesOfARowFieldForThePluginTheTokenNames(t *testing.T) {
	rows := &meadowRows{}
	client := newRowsClient(t, rows, "token-b")

	response, err := client.ListRowOptions(t.Context(), connect.NewRequest(&v1.HostConnectionServiceListRowOptionsRequest{
		Operation: "llm.chat", ConnectionId: "c-meadow", Field: "model",
	}))

	if err != nil {
		t.Fatal(err)
	}
	options := response.Msg.GetOptions()
	if rows.plugin != "md.numen.b" || len(options) != 1 || options[0].GetValue() != "wren-2" || options[0].GetTitle() != "Wren 2" {
		t.Errorf("plugin = %q, options = %v", rows.plugin, options)
	}
}

func TestListRowOptionsOfAConnectionThatIsNotAmongTheRowsIsNotFound(t *testing.T) {
	client := newRowsClient(t, &meadowRows{}, "token-b")

	_, err := client.ListRowOptions(t.Context(), connect.NewRequest(&v1.HostConnectionServiceListRowOptionsRequest{
		Operation: "llm.chat", ConnectionId: "c-elsewhere", Field: "model",
	}))

	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("code = %v, want not found", connect.CodeOf(err))
	}
}

func TestListRowsWithoutAKnownTokenIsUnauthenticated(t *testing.T) {
	rows := &meadowRows{}
	client := newRowsClient(t, rows, "token-unknown")

	_, err := client.ListRows(t.Context(), connect.NewRequest(&v1.HostConnectionServiceListRowsRequest{Operation: "llm.chat"}))

	if connect.CodeOf(err) != connect.CodeUnauthenticated || len(rows.asked) != 0 {
		t.Errorf("code = %v, asked = %v", connect.CodeOf(err), rows.asked)
	}
}

func TestAHostWithNoRowsListsNone(t *testing.T) {
	client := newRowsClient(t, nil, "token-b")

	response, err := client.ListRows(t.Context(), connect.NewRequest(&v1.HostConnectionServiceListRowsRequest{Operation: "llm.chat"}))

	if err != nil || len(response.Msg.GetRows()) != 0 {
		t.Errorf("rows = %v, %v, want none", response, err)
	}
}

func TestACallCarriesThePreferredRowOfTheRequest(t *testing.T) {
	calls := &scriptedCalls{done: "finish"}
	client, _ := newConnectionClient(t, calls, "token-b")

	_, err := call(t.Context(), client, &v1.HostConnectionServiceCallRequest{
		Operation: "llm.chat", Preferred: &v1.PreferredRow{ConnectionId: "c-meadow", Options: map[string]string{"model": "wren-2"}},
	})

	preferred := calls.request.Preferred
	if err != nil || preferred == nil || preferred.ConnectionID != "c-meadow" || preferred.Options["model"] != "wren-2" {
		t.Fatalf("preferred = %+v, %v", preferred, err)
	}
}
