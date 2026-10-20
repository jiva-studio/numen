package plugin

import (
	"errors"
	"io"
	"testing"

	"connectrpc.com/connect"
)

func TestReadRPCErrorNamesTheCodeOnce(t *testing.T) {
	inner := connect.NewError(connect.CodeUnavailable, io.ErrUnexpectedEOF)
	outer := connect.NewError(connect.CodeUnavailable, inner)

	err := readRPCError("embed batch rpc", outer)

	if want := "embed batch rpc: unavailable: unexpected EOF"; err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal("the cause is not reachable from the error")
	}
}

func TestReadRPCErrorKeepsADifferentCodeOfTheMessage(t *testing.T) {
	err := readRPCError("embed batch rpc", connect.NewError(connect.CodeInternal, errors.New("unavailable: model")))

	if want := "embed batch rpc: internal: unavailable: model"; err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestReadRPCErrorWrapsAnErrorThatIsNoConnectError(t *testing.T) {
	cause := errors.New("connection refused")

	err := readRPCError("embed batch rpc", cause)

	if want := "embed batch rpc: connection refused"; err.Error() != want || !errors.Is(err, cause) {
		t.Fatalf("got %q", err.Error())
	}
}
