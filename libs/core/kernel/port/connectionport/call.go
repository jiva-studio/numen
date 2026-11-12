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
}
