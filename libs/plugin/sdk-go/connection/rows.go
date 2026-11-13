package connection

import (
	"context"
	"fmt"
	"maps"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// PreferredRow is a connection among the rows of the plugin for the operation, asked before the other rows
// with Options in place of the values of its row fields. A connection that is not among the rows is
// ignored.
//
//numen:api
type PreferredRow struct {
	// ConnectionID is the id of the connection, as a Row lists it.
	ConnectionID string
	// Options are the values of the row fields for the call.
	Options map[string]string
}

// Row is one row of the plugin: its connection, the values of its row fields, and the row fields the type
// of the connection declares.
//
//numen:api
type Row struct {
	// ConnectionID is the id of the connection the row names.
	ConnectionID string
	// ConnectionName is the name the person reads.
	ConnectionName string
	// Options are the values of the row fields by key.
	Options map[string]string
	// Fields are the row fields of the type, in the order of the type.
	Fields []RowField
}

// RowField is a row field a connection type declares.
//
//numen:api
type RowField struct {
	// Key is the key of the value in Options.
	Key string
	// Title is what the person reads.
	Title string
	// OptionsFrom is the operation that lists the values of the field, empty for none.
	OptionsFrom string
}

type preferredRowKey struct{}

// WithPreferredRow returns a context whose calls ask the row first.
func WithPreferredRow(ctx context.Context, row PreferredRow) context.Context {
	return context.WithValue(ctx, preferredRowKey{}, row)
}

func readPreferredRow(ctx context.Context) *v1.PreferredRow {
	row, isSet := ctx.Value(preferredRowKey{}).(PreferredRow)
	if !isSet {
		return nil
	}
	return &v1.PreferredRow{ConnectionId: row.ConnectionID, Options: maps.Clone(row.Options)}
}

// ListRows returns the rows the plugin calls the operation through, in the order of use. A row whose
// connection is missing, or whose type is unavailable or does not serve the operation, is left out.
func (c *Client) ListRows(ctx context.Context, operation string) ([]Row, error) {
	response, err := c.host.ListRows(ctx, connect.NewRequest(&v1.HostConnectionServiceListRowsRequest{Operation: operation}))
	if err != nil {
		return nil, fmt.Errorf("connection: list the rows of %s: %w", operation, err)
	}
	wired := response.Msg.GetRows()
	rows := make([]Row, len(wired))
	for i, one := range wired {
		fields := make([]RowField, len(one.GetFields()))
		for j, field := range one.GetFields() {
			fields[j] = RowField{Key: field.GetKey(), Title: field.GetTitle(), OptionsFrom: field.GetOptionsFrom()}
		}
		rows[i] = Row{
			ConnectionID: one.GetConnectionId(), ConnectionName: one.GetConnectionName(), Options: maps.Clone(one.GetOptions()), Fields: fields,
		}
	}
	return rows, nil
}

// ListRowOptions returns the values a row field of a connection among the rows of the operation can take,
// as its provider lists them for options. A connection that is not among the rows fails with the code not
// found.
func (c *Client) ListRowOptions(ctx context.Context, operation, connectionID, field string, options map[string]string) ([]Option, error) {
	response, err := c.host.ListRowOptions(ctx, connect.NewRequest(&v1.HostConnectionServiceListRowOptionsRequest{
		Operation: operation, ConnectionId: connectionID, Field: field, Options: options,
	}))
	if err != nil {
		return nil, fmt.Errorf("connection: list the values of %s: %w", field, err)
	}
	wired := response.Msg.GetOptions()
	listed := make([]Option, len(wired))
	for i, one := range wired {
		listed[i] = Option{Value: one.GetValue(), Title: one.GetTitle(), Group: one.GetGroup()}
	}
	return listed, nil
}
