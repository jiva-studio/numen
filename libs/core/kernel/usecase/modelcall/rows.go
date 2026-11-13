package modelcall

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// ListRows returns the rows of the plugin for the operation in the order of use, each with the name of its
// connection and the row fields of its type. A row whose connection is missing, or whose type is
// unavailable or does not serve the operation, is left out.
func (c *Calls) ListRows(callerPluginID, operation string) ([]modelcall.Row, error) {
	rows, err := c.rows(callerPluginID, operation)
	if err != nil {
		return nil, err
	}
	listed := []modelcall.Row{}
	for _, row := range rows {
		resolved, err := c.resolve(row.ConnectionID)
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
func (c *Calls) ListRowOptions(
	ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string,
) ([]modelcall.Option, error) {
	rows, err := c.rows(callerPluginID, operation)
	if err != nil {
		return nil, err
	}
	if !slices.ContainsFunc(rows, func(row connection.Row) bool { return row.ConnectionID == connectionID }) {
		return nil, connection.ErrNotFound
	}
	return c.listOptions(ctx, connectionID, field, options, []string{operation})
}
