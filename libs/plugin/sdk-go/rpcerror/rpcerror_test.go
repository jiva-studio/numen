package rpcerror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
)

func TestTheCodeOfAnErrorIsReadBack(t *testing.T) {
	cause := errors.New("no model")
	err := fmt.Errorf("embed: %w", rpcerror.New(rpcerror.FailedPrecondition, cause))
	if got := rpcerror.GetCode(err); got != rpcerror.FailedPrecondition {
		t.Errorf("code %v, want %v", got, rpcerror.FailedPrecondition)
	}
	if !errors.Is(err, cause) {
		t.Error("the cause is not reachable through the error")
	}
}

func TestAPlainErrorCarriesNoCodeOfTheList(t *testing.T) {
	if got := rpcerror.GetCode(errors.New("plain")); got == rpcerror.Internal {
		t.Errorf("a plain error reads as %v", got)
	}
}
