package delivery

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
)

func TestOnlyAnEndedContextOrATimeoutIsAFailureOfTheCaller(t *testing.T) {
	if !isCallerFailure(t.Context(), connect.NewError(connect.CodeDeadlineExceeded, errors.New("late"))) {
		t.Error("a timeout is not a failure of the caller")
	}
	if !isCallerFailure(t.Context(), connect.NewError(connect.CodeCanceled, errors.New("stopped"))) {
		t.Error("a cancel is not a failure of the caller")
	}
	if isCallerFailure(t.Context(), connect.NewError(connect.CodeInternal, errors.New("broken"))) {
		t.Error("an answer of the plugin is a failure of the caller")
	}
}
