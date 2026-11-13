package wire

import (
	"context"
	"errors"
	"maps"

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
			failures[i] = &v1.CallRowFailure{ConnectionName: one.ConnectionName, Reason: one.Reason, Code: convertFailureReasonToWire(one.Code)}
		}
		return &v1.CallFailure{Error: v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED, Detail: allFailed.Error(), Failures: failures}, true
	case errors.As(err, &stopped):
		return &v1.CallFailure{
			Error: v1.CallErrorCode_CALL_ERROR_CODE_STOPPED, Detail: stopped.Error(),
			Failures: []*v1.CallRowFailure{{
				ConnectionName: stopped.ConnectionName, Reason: stopped.Reason, Code: convertFailureReasonToWire(stopped.Code),
			}},
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

// ConvertPreferredRowFromWire returns the row a call asks first, nil where the request names none.
func ConvertPreferredRowFromWire(preferred *v1.PreferredRow) *modelcall.PreferredRow {
	if preferred.GetConnectionId() == "" {
		return nil
	}
	return &modelcall.PreferredRow{
		ConnectionID: preferred.GetConnectionId(), Options: maps.Clone(preferred.GetOptions()), IsExclusive: preferred.GetIsExclusive(),
	}
}

// ConvertCallRowsToWire returns the rows of a plugin as the call services list them.
func ConvertCallRowsToWire(rows []modelcall.Row) []*v1.CallRow {
	wired := make([]*v1.CallRow, len(rows))
	for i, row := range rows {
		fields := make([]*v1.CallRowField, len(row.Fields))
		for j, one := range row.Fields {
			fields[j] = &v1.CallRowField{Key: one.Key, Title: one.Title, OptionsFrom: one.OptionsFrom, IsRequired: one.IsRequired}
		}
		wired[i] = &v1.CallRow{
			ConnectionId: row.ConnectionID, ConnectionName: row.ConnectionName, Options: maps.Clone(row.Options), Fields: fields,
		}
	}
	return wired
}

// ConvertOptionsToWire returns the values of a row field as the call services list them.
func ConvertOptionsToWire(options []modelcall.Option) []*v1.Option {
	wired := make([]*v1.Option, len(options))
	for i, one := range options {
		wired[i] = &v1.Option{Value: one.Value, Title: one.Title, Group: one.Group}
	}
	return wired
}

// NewRowListError returns the status of a listing of rows or of the values of a row field that failed: not
// found for a connection that is not among the rows, canceled where its context ended, and unavailable
// with the reason otherwise.
func NewRowListError(ctx context.Context, err error) error {
	switch {
	case ctx.Err() != nil:
		return connect.NewError(connect.CodeCanceled, ctx.Err())
	case errors.Is(err, connection.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeUnavailable, err)
}

func convertFailureReasonToWire(reason connection.FailureReason) v1.FailureReason {
	switch reason {
	case connection.FailureReasonRejectedKey:
		return v1.FailureReason_FAILURE_REASON_REJECTED_KEY
	case connection.FailureReasonPaymentRequired:
		return v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED
	case connection.FailureReasonRateLimited:
		return v1.FailureReason_FAILURE_REASON_RATE_LIMITED
	case connection.FailureReasonInvalidRequest:
		return v1.FailureReason_FAILURE_REASON_INVALID_REQUEST
	case connection.FailureReasonUnavailable:
		return v1.FailureReason_FAILURE_REASON_UNAVAILABLE
	}
	return v1.FailureReason_FAILURE_REASON_UNAVAILABLE
}
