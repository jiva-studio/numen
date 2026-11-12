// Package rpcerror gives a plugin the error codes of the calls it serves and makes, so that a
// plugin names no RPC library of its own.
package rpcerror

import "connectrpc.com/connect"

// Code is the code of a failed call, as its client reads it.
type Code = connect.Code

// The codes a plugin answers with or checks for.
const (
	Canceled           = connect.CodeCanceled
	DeadlineExceeded   = connect.CodeDeadlineExceeded
	FailedPrecondition = connect.CodeFailedPrecondition
	Internal           = connect.CodeInternal
	Unavailable        = connect.CodeUnavailable
	Unauthenticated    = connect.CodeUnauthenticated
)

// New returns err as the failure of a call with the given code.
func New(code Code, err error) error {
	return connect.NewError(code, err)
}

// GetCode returns the code a failed call carries: Unknown for an error that carries none, and 0
// for nil.
func GetCode(err error) Code {
	return connect.CodeOf(err)
}
