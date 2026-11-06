package transport

import (
	"context"
	"maps"
	"strings"
	"testing"
	"time"
)

func TestParseReadyLineReadsTheListenerPorts(t *testing.T) {
	handshake, ok := ParseHandshake("NUMEN_READY port=41234 listener.mcp=41235 listener.debug-console=41236")
	if !ok {
		t.Fatal("the ready line was not read as a handshake")
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
	handshake, ok := ParseHandshake("NUMEN_READY port=41234")
	if !ok {
		t.Fatal("the ready line was not read as a handshake")
	}

	if handshake.Port != 41234 || len(handshake.Listeners) != 0 {
		t.Errorf("handshake = %+v, want port 41234 and no listeners", handshake)
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
