package process

import (
	"strings"
	"testing"
)

func TestTheStandardErrorKeepsOnlyItsEnd(t *testing.T) {
	var stderr tailBuffer
	for range 3 {
		if _, err := stderr.Write([]byte(strings.Repeat("a", maxStderr/2))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := stderr.Write([]byte("the token was refused")); err != nil {
		t.Fatal(err)
	}

	kept := stderr.String()

	if len(kept) != maxStderr || !strings.HasSuffix(kept, "the token was refused") {
		t.Errorf("kept %d bytes ending in %q, want %d ending in the last line", len(kept), kept[len(kept)-30:], maxStderr)
	}
}
