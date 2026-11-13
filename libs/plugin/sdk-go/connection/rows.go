package connection

import (
	"context"
	"fmt"
	"maps"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// RowField is a row field a connection type declares. OptionsFrom is the operation that lists its values,
// empty for none. IsRequired is a field a row must hold a value for.
type RowField struct {
	Key         string
	Title       string
	OptionsFrom string
	IsRequired  bool
}

// Row is one row of the plugin: its connection, the values of its row fields by key, and the row fields
// the type of the connection declares.
type Row struct {
	ConnectionID   string
	ConnectionName string
	Options        map[string]string
	Fields         []RowField
}

// RowOptionsRequest names a row field of a connection among the rows of an operation, with the values of
// the row fields so far.
type RowOptionsRequest struct {
	Operation    string
	ConnectionID string
	Field        string
	Options      map[string]string
}

// PreferredRow is a connection among the rows of the plugin for the operation, asked before the other rows
// with Options in place of the values of its row fields. A connection that is not among the rows is
// ignored.
type PreferredRow struct {
	ConnectionID string
	Options      map[string]string
}

// ListRows returns the rows of the plugin for the operation in the order of use, without the rows whose
// connection is missing or whose type is unavailable or does not serve the operation.
func (c *Client) ListRows(ctx context.Context, operation string) ([]Row, error) {
	response, err := c.host.ListRows(ctx, connect.NewRequest(&v1.HostConnectionServiceListRowsRequest{Operation: operation}))
	if err != nil {
		return nil, readListError(ctx, "list the rows of "+operation, err)
	}
	rows := make([]Row, len(response.Msg.GetRows()))
	for i, one := range response.Msg.GetRows() {
		fields := make([]RowField, len(one.GetFields()))
		for j, field := range one.GetFields() {
			fields[j] = RowField{Key: field.GetKey(), Title: field.GetTitle(), OptionsFrom: field.GetOptionsFrom(), IsRequired: field.GetIsRequired()}
		}
		rows[i] = Row{
			ConnectionID: one.GetConnectionId(), ConnectionName: one.GetConnectionName(), Options: maps.Clone(one.GetOptions()), Fields: fields,
		}
	}
	return rows, nil
}

// ListRowOptions returns the values a row field of a connection among the rows of the operation can take,
// as its provider lists them for the options of the request. It fails for a connection that is not among
// the rows and for a provider that cannot list them.
func (c *Client) ListRowOptions(ctx context.Context, request RowOptionsRequest) ([]Option, error) {
	response, err := c.host.ListRowOptions(ctx, connect.NewRequest(&v1.HostConnectionServiceListRowOptionsRequest{
		Operation: request.Operation, ConnectionId: request.ConnectionID, Field: request.Field, Options: maps.Clone(request.Options),
	}))
	if err != nil {
		return nil, readListError(ctx, "list the values of "+request.Field, err)
	}
	options := make([]Option, len(response.Msg.GetOptions()))
	for i, one := range response.Msg.GetOptions() {
		options[i] = Option{Value: one.GetValue(), Title: one.GetTitle(), Group: one.GetGroup()}
	}
	return options, nil
}

// Prefer returns a Caller that asks row before the other rows of the plugin.
func (c *Client) Prefer(row PreferredRow) Caller {
	return &preferringClient{client: c, row: row}
}

type preferringClient struct {
	client *Client
	row    PreferredRow
}

func (p *preferringClient) Call(ctx context.Context, operation string, payload []byte, priority Priority) (Frames, error) {
	return p.client.call(ctx, &v1.HostConnectionServiceCallRequest{
		Operation: operation, Payload: payload, Priority: writePriority(priority),
		Preferred: &v1.PreferredRow{ConnectionId: p.row.ConnectionID, Options: maps.Clone(p.row.Options)},
	})
}

func readListError(ctx context.Context, what string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("connection: %s: %w", what, err)
}
