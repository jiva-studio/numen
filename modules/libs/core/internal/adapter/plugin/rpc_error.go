package plugin

import (
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
)

// rpcError is a failed call that reads with the code of its Connect error once.
type rpcError struct {
	text string
	err  error
}

func (e *rpcError) Error() string { return e.text }

func (e *rpcError) Unwrap() error { return e.err }

// readRPCError names the failure of a call. A Connect error reads as its code followed by its
// message, with a code the message repeats at its start dropped.
func readRPCError(call string, err error) error {
	var failure *connect.Error
	if !errors.As(err, &failure) {
		return fmt.Errorf("%s: %w", call, err)
	}
	prefix := failure.Code().String() + ": "
	message := failure.Message()
	for strings.HasPrefix(message, prefix) {
		message = strings.TrimPrefix(message, prefix)
	}
	return &rpcError{text: call + ": " + prefix + message, err: err}
}
