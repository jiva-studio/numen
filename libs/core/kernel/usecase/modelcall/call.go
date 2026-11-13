package modelcall

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// Call asks the rows of the plugin for the operation, in order, and sends the data frames to send as they
// arrive. The slot of the connection and the provider are held until the call ends. A row that fails before
// its first data frame is set aside and the next row is asked. A row that fails after it ends the call with
// *connection.AllFailedError naming that connection and the reason, and no call is repeated. It returns the
// done payload, connection.ErrNoConnection where the plugin has no row for the operation,
// *connection.AllFailedError where every row failed, *connection.StoppedError where a connection found the
// request wrong, the error of send where send fails, and the error of the context where the call was
// canceled.
func (c *Calls) Call(ctx context.Context, callerPluginID string, request modelcall.Request, send modelcall.Send) ([]byte, error) {
	isStarted := false
	return carry(ctx, c, asking{callerPluginID, request.Operation, request.Priority, request.Preferred}, func(
		ctx context.Context, provider connectionport.Provider, call modelcall.ProviderCall,
	) ([]byte, error) {
		var sendErr error
		done, err := provider.Call(ctx, request.Operation, call, request.Payload, func(data []byte) error {
			isStarted = true
			sendErr = send(data)
			return sendErr
		})
		switch {
		case err == nil:
			return done, nil
		case sendErr != nil:
			return nil, &consumerError{err: sendErr}
		case isStarted:
			return nil, &midStreamError{err: err}
		}
		return nil, err
	})
}

// consumerError is a send that failed: the consumer went away, and no connection is at fault.
type consumerError struct{ err error }

func (e *consumerError) Error() string { return e.err.Error() }

func (e *consumerError) Unwrap() error { return e.err }

// midStreamError is a provider that failed after it sent its first data frame.
type midStreamError struct{ err error }

func (e *midStreamError) Error() string { return e.err.Error() }

func (e *midStreamError) Unwrap() error { return e.err }

func readMidStreamVerdict(name string, err error) verdict {
	reason, retryAfter, isTryNext := connection.FailureReasonUnavailable, time.Duration(0), false
	var stopError *connection.StopError
	var tryNext *connection.TryNextError
	switch {
	case errors.As(err, &stopError):
		reason = stopError.Reason
	case errors.As(err, &tryNext):
		reason, retryAfter, isTryNext = tryNext.Reason, tryNext.RetryAfter, true
	}
	return verdict{
		err:    &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: name, Reason: reason.String(), Code: reason}}},
		reason: reason, retryAfter: retryAfter, isCooling: isTryNext,
	}
}
