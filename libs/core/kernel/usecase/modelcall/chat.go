package modelcall

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
)

// Chat asks the rows of the plugin for the operation chat, in order, and sends the events of the turn to
// send as they arrive. The slot of the connection and the provider are held until the turn ends. A row that
// fails before its first event is set aside and the next row is asked. A row that fails after it ends the
// call with *connection.AllFailedError naming that connection and the reason, and no call is repeated.
// It returns the other errors Complete does, and the error of send where send fails.
func (c *Calls) Chat(
	ctx context.Context, callerPluginID string, request modelcall.ChatRequest, priority modelcall.Priority,
	send func(modelcall.ChatEvent) error,
) error {
	isStarted := false
	_, err := carry(ctx, c, asking{callerPluginID, "chat", priority}, pickChat, func(
		ctx context.Context, provider connectionport.ChatProvider, call modelcall.ProviderCall, _ connection.Row,
	) (struct{}, error) {
		var sendErr error
		err := provider.Chat(ctx, call, request, func(event modelcall.ChatEvent) error {
			isStarted = true
			sendErr = send(event)
			return sendErr
		})
		switch {
		case err == nil:
			return struct{}{}, nil
		case sendErr != nil:
			return struct{}{}, &consumerError{err: sendErr}
		case isStarted:
			return struct{}{}, &midStreamError{err: err}
		}
		return struct{}{}, err
	})
	return err
}

// consumerError is a send that failed: the consumer went away, and no connection is at fault.
type consumerError struct{ err error }

func (e *consumerError) Error() string { return e.err.Error() }

func (e *consumerError) Unwrap() error { return e.err }

// midStreamError is a provider that failed after it sent its first event.
type midStreamError struct{ err error }

func (e *midStreamError) Error() string { return e.err.Error() }

func (e *midStreamError) Unwrap() error { return e.err }

func readMidStreamVerdict(name string, err error) verdict {
	reason, retryAfter, isTryNext := connection.FailureReasonUnreachable, time.Duration(0), false
	var stopError *connection.StopError
	var tryNext *connection.TryNextError
	switch {
	case errors.As(err, &stopError):
		reason = stopError.Reason
	case errors.As(err, &tryNext):
		reason, retryAfter, isTryNext = tryNext.Reason, tryNext.RetryAfter, true
	}
	return verdict{
		err:    &connection.AllFailedError{Failures: []connection.Failure{{ConnectionName: name, Reason: reason.String()}}},
		reason: reason, retryAfter: retryAfter, isCooling: isTryNext,
	}
}

func pickChat(provider connectionport.Provider) connectionport.ChatProvider { return provider }
