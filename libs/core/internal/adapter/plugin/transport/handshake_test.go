package transport

import (
	"context"
	"errors"
	"io"
	"maps"
	"strings"
	"testing"
	"time"
)

func TestParseReadyLineReadsTheListenerPorts(t *testing.T) {
	handshake, ok, err := ParseHandshake("NUMEN_READY port=41234 listener.mcp=41235 listener.debug-console=41236")
	if err != nil || !ok {
		t.Fatalf("the ready line was not read: ok %v, error %v", ok, err)
	}

	if handshake.Port != 41234 {
		t.Errorf("port = %d, want 41234", handshake.Port)
	}
	want := map[string]int{"mcp": 41235, "debug-console": 41236}
	if !maps.Equal(handshake.Listeners, want) {
		t.Errorf("listeners = %v, want %v", handshake.Listeners, want)
	}
}

func TestParseReadyLineReadsNoListenersFromALineWithoutThem(t *testing.T) {
	handshake, ok, err := ParseHandshake("NUMEN_READY port=41234")
	if err != nil || !ok {
		t.Fatalf("the ready line was not read: ok %v, error %v", ok, err)
	}

	if handshake.Port != 41234 || len(handshake.Listeners) != 0 {
		t.Errorf("handshake = %+v, want port 41234 and no listeners", handshake)
	}
}

func TestParseReadyLineRejectsAMalformedListenerToken(t *testing.T) {
	for name, token := range map[string]string{
		"no name":                          "listener.=41235",
		"a port that is text":              "listener.mcp=open",
		"port 0":                           "listener.mcp=0",
		"a negative port":                  "listener.mcp=-41235",
		"a fractional port":                "listener.mcp=41235.5",
		"no port":                          "listener.mcp=",
		"no equals sign":                   "listener.mcp",
		"a name given twice":               "listener.mcp=41235 listener.mcp=41236",
		"a name given twice with one port": "listener.mcp=41235 listener.debug=41237 listener.mcp=41235",
	} {
		_, ok, err := ParseHandshake("NUMEN_READY port=41234 " + token)

		if err == nil {
			t.Errorf("%s: ParseHandshake(%q) returned no error (ok %v)", name, token, ok)
		}
	}
}

func TestParseReadyLineReturnsNoErrorForNonReadyLine(t *testing.T) {
	_, ok, err := ParseHandshake("starting listener.mcp=none")

	if ok || err != nil {
		t.Errorf("ParseHandshake = ok %v, error %v; want false and no error", ok, err)
	}
}

func TestWaitForReadyLineReturnsTheListenerPorts(t *testing.T) {
	input := "starting\nNUMEN_READY port=41234 listener.mcp=41235\nserving\n"
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	handshake, err := WaitForHandshake(ctx, strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	if handshake.Port != 41234 {
		t.Errorf("port = %d, want 41234", handshake.Port)
	}
	if !maps.Equal(handshake.Listeners, map[string]int{"mcp": 41235}) {
		t.Errorf("listeners = %v, want mcp=41235", handshake.Listeners)
	}
}

func TestWaitForReadyLineFailsAtOnceOnMalformedListenerToken(t *testing.T) {
	output, writer := io.Pipe()
	t.Cleanup(func() {
		if err := writer.Close(); err != nil {
			t.Errorf("close the pipe: %v", err)
		}
	})
	go func() {
		if _, err := io.WriteString(writer, "NUMEN_READY port=41234 listener.mcp=open\n"); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("write the ready line: %v", err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	_, err := WaitForHandshake(ctx, output)

	if err == nil {
		t.Fatal("WaitForHandshake returned no error for a malformed listener token")
	}
	if errors.Is(err, ErrHandshakeTimeout) {
		t.Errorf("error = %v, want the error of the token before the timeout", err)
	}
}
