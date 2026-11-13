package connectionport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// ModelCalls carries out the operations plugins ask for. The connections come from the rows of the
// setting of the calling plugin that is used for the operation, and a request names none. At most as many
// calls as the field parallelRequests of a connection run at once; the rest wait, the interactive ones
// first, and a wait that times out sends the call to the next row.
type ModelCalls interface {
	// Call asks the rows in order and sends the data frames of the row that answers to send as they
	// arrive. A row that fails before its first data frame is set aside and the next row is asked; a row
	// that fails after it ends the call with *connection.AllFailedError naming that connection, and nothing
	// is repeated. It returns the done payload, or connection.ErrNoConnection, *connection.AllFailedError
	// or *connection.StoppedError where no row answers, the error of send when send fails, and the error
	// of the context when the call is canceled.
	Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error)
	RowLists
}

// RowLists lists the rows plugins call operations through, for a plugin to offer a choice of them.
type RowLists interface {
	// ListRows returns the rows of the plugin for the operation in the order of use, without the rows whose
	// connection is missing or whose type is unavailable or does not serve the operation.
	ListRows(callerPluginID, operation string) ([]modelcall.Row, error)
	// ListRowOptions returns the values a row field of a connection among those rows can take, as its
	// provider lists them for options. It fails with connection.ErrNotFound for a connection that is not
	// among the rows, and with the errors of ListRowOptions.
	ListRowOptions(ctx context.Context, callerPluginID, operation, connectionID, field string, options map[string]string) ([]modelcall.Option, error)
}

// ListRowOptions returns the values a row field of a connection can take, as its provider lists them for the
// options of the row and the operations of its setting.
type ListRowOptions func(ctx context.Context, connectionID, field string, options map[string]string, operations []string) ([]modelcall.Option, error)
