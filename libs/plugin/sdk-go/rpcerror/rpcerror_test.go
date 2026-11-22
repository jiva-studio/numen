package rpcerror_test

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
)

func TestAnErrorCarriesItsCodeAndItsCause(t *testing.T) {
	cause := errors.New("no model")
	err := fmt.Errorf("embed: %w", rpcerror.New(rpcerror.FailedPrecondition, cause))
	if got := connect.CodeOf(err); got != connect.CodeFailedPrecondition {
		t.Errorf("code %v, want %v", got, connect.CodeFailedPrecondition)
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is not reachable through the error")
	}
}

func TestEveryCodeReadsAsTheCodeOfTheClient(t *testing.T) {
	for code, want := range map[rpcerror.Code]connect.Code{
		rpcerror.Canceled:           connect.CodeCanceled,
		rpcerror.DeadlineExceeded:   connect.CodeDeadlineExceeded,
		rpcerror.FailedPrecondition: connect.CodeFailedPrecondition,
		rpcerror.Internal:           connect.CodeInternal,
		rpcerror.Unavailable:        connect.CodeUnavailable,
		rpcerror.Unauthenticated:    connect.CodeUnauthenticated,
	} {
		if got := connect.CodeOf(rpcerror.New(code, errors.New("failed"))); got != want {
			t.Errorf("%v reads as %v, want %v", code, got, want)
		}
	}
}
