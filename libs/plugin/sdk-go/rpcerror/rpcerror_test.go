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
	if got := connect.CodeOf(err); got != rpcerror.FailedPrecondition {
		t.Errorf("code %v, want %v", got, rpcerror.FailedPrecondition)
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is not reachable through the error")
	}
}
