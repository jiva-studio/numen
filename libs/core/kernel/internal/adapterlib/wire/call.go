package wire

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// ConvertCallFailureToWire returns the failure frame of a call the core could not serve, and false for an error
// that is no such failure or a call whose context ended.
func ConvertCallFailureToWire(ctx context.Context, err error) (*v1.CallFailure, bool) {
	if ctx.Err() != nil {
		return nil, false
	}
	var allFailed *connection.AllFailedError
	var stopped *connection.StoppedError
	switch {
	case errors.Is(err, connection.ErrNoConnection):
		return &v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION}, true
	case errors.As(err, &allFailed):
		failures := make([]*v1.CallRowFailure, len(allFailed.Failures))
		for i, one := range allFailed.Failures {
			failures[i] = &v1.CallRowFailure{ConnectionName: one.ConnectionName, Reason: one.Reason}
		}
		return &v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED, Detail: allFailed.Error(), Failures: failures}, true
	case errors.As(err, &stopped):
		return &v1.CallFailure{
			Error: v1.CallErrorCode_CALL_ERROR_CODE_STOPPED, Detail: stopped.Error(),
			Failures: []*v1.CallRowFailure{{ConnectionName: stopped.ConnectionName, Reason: stopped.Reason}},
		}, true
	}
	return nil, false
}

// NewCallStatusError returns the status of a call that ends without a failure frame: canceled where its context
// ended, internal otherwise.
func NewCallStatusError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}

// ConvertPriorityFromWire returns the priority of a call; an unspecified priority is background.
func ConvertPriorityFromWire(priority v1.Priority) modelcall.Priority {
	if priority == v1.Priority_PRIORITY_INTERACTIVE {
		return modelcall.PriorityInteractive
	}
	return modelcall.PriorityBackground
}
