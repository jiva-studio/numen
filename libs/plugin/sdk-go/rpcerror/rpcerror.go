// Package rpcerror gives a plugin the error codes of the calls it serves, so that a
// plugin names no RPC library of its own.
package rpcerror

import "connectrpc.com/connect"

// Code is the code of a failed call, as its client reads it.
type Code int

// The codes a plugin answers with.
const (
	Canceled Code = iota + 1
	DeadlineExceeded
	FailedPrecondition
	Internal
	Unavailable
	Unauthenticated
)

// String returns the name the client reads for the code.
func (c Code) String() string { return getConnectCode(c).String() }

// New returns err as the failure of a call with the given code.
func New(code Code, err error) error {
	return connect.NewError(getConnectCode(code), err)
}

func getConnectCode(code Code) connect.Code {
	switch code {
	case Canceled:
		return connect.CodeCanceled
	case DeadlineExceeded:
		return connect.CodeDeadlineExceeded
	case FailedPrecondition:
		return connect.CodeFailedPrecondition
	case Internal:
		return connect.CodeInternal
	case Unavailable:
		return connect.CodeUnavailable
	case Unauthenticated:
		return connect.CodeUnauthenticated
	}
	return connect.CodeUnknown
}
