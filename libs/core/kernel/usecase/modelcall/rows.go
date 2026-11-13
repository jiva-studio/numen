package modelcall

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// Rows lists the rows plugins call operations through, for a plugin view to offer a choice of them.
type Rows struct {
	resolve     connectionport.ResolveConnection
	rows        connectionport.ListRows
	listOptions connectionport.ListRowOptions
}

// NewRows returns the listing over the rows of the plugins, the connections they name and the providers
// that list the values of their row fields.
func NewRows(resolve connectionport.ResolveConnection, rows connectionport.ListRows, listOptions connectionport.ListRowOptions) *Rows {
	return &Rows{resolve: resolve, rows: rows, listOptions: listOptions}
}

// ListRows returns the rows of the plugin for the operation in the order of use, each with the name of its
// connection and the row fields of its type. A row whose connection is missing, or whose type is
// unavailable or does not serve the operation, is left out.
func (r *Rows) ListRows(callerPluginID, operation string) ([]modelcall.Row, error) {
	rows, err := r.rows(callerPluginID, operation)
	if err != nil {
		return nil, err
	}
	listed := []modelcall.Row{}
	for _, row := range rows {
		resolved, err := r.resolve(row.ConnectionID)
		switch {
		case errors.Is(err, connection.ErrNotFound), errors.Is(err, connection.ErrTypeUnavailable):
			continue
		case err != nil:
			return nil, err
		}
		if !resolved.Declaration.HasOperation(operation) {
			continue
		}
		fields := make([]modelcall.RowField, len(resolved.Declaration.RowFields))
		for i, one := range resolved.Declaration.RowFields {
			fields[i] = modelcall.RowField{Key: one.Key, Title: one.Title, OptionsFrom: one.OptionsFrom}
		}
		listed = append(listed, modelcall.Row{
			ConnectionID: resolved.ID, ConnectionName: resolved.Name, Options: maps.Clone(row.Options), Fields: fields,
		})
	}
	return listed, nil
}

// ListRowOptions returns the values the row field of the connection can take, as its provider lists them for
// options. It fails with connection.ErrNotFound where the connection is not among the rows of the plugin for
// the operation.
func (r *Rows) ListRowOptions(
	ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string,
) ([]modelcall.Option, error) {
	rows, err := r.rows(callerPluginID, operation)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(rows, func(row connection.Row) bool { return row.ConnectionID == connectionID }) {
		return nil, connection.ErrNotFound
	}
	return r.listOptions(ctx, connectionID, field, options, []string{operation})
}
