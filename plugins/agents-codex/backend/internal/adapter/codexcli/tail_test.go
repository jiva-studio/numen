package codexcli

import (
	"strings"
	"testing"
)

func TestTheStandardErrorKeepsOnlyItsEnd(t *testing.T) {
	if maxStderr != 64*1024 {
		t.Fatalf("maxStderr = %d, want 65536", maxStderr)
	}
	var stderr tailBuffer
	if n, err := stderr.Write([]byte("short")); err != nil || n != 5 || stderr.String() != "short" {
		t.Fatalf("short write = (%d, %v), %q", n, err, stderr.String())
	}
	for range 3 {
		chunk := []byte(strings.Repeat("a", maxStderr/2))
		if n, err := stderr.Write(chunk); err != nil || n != len(chunk) {
			t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(chunk))
		}
	}
	suffix := []byte("the token was refused")
	if n, err := stderr.Write(suffix); err != nil || n != len(suffix) {
		t.Fatalf("Write = (%d, %v), want (%d, nil)", n, err, len(suffix))
	}

	kept := stderr.String()

	if len(kept) != maxStderr || !strings.HasSuffix(kept, "the token was refused") {
		t.Errorf("kept %d bytes ending in %q, want %d ending in the last line", len(kept), kept[len(kept)-30:], maxStderr)
	}
}
