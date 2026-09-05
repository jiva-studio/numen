package mcp

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The fingerprint an agent is handed and hands back is the size and the
// modification time in nanoseconds since the epoch, and no other number.
func TestTheFingerprintAnAgentCarriesIsSizeAndNanoseconds(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	ref := domain.Fingerprint{Path: "notes/Compost.md", Size: 12, ModTime: at}

	want := "12-1788433445123456789"
	if held := fingerprintOf(ref); held != want {
		t.Errorf("the agent is handed %q, want %q", held, want)
	}
}

// A stamp a clock handed out carries a monotonic reading, and the string an
// agent hands back carries none. The note is still the note that was read.
func TestAFingerprintTakenFromAClockComesBackAsTheSameFile(t *testing.T) {
	ref := domain.Fingerprint{Path: "notes/Compost.md", Size: 12, ModTime: time.Now()}

	back, err := parseFingerprint(fingerprintOf(ref))
	if err != nil {
		t.Fatal(err)
	}
	if !back.Unchanged(ref) {
		t.Errorf("the agent handed back %+v, and the note is %+v", back, ref)
	}
}
