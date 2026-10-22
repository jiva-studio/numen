package editor

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// An address and a message both carry a modification time as nanoseconds since
// the epoch, and as that number and no other.
func TestTheSchemaAndTheAddressCarryNanosecondsSinceTheEpoch(t *testing.T) {
	pos := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	const nanos = 1788433445123456789

	ref := domain.Fingerprint{Path: "library/Compost.pdf", Size: 12, ModTime: pos}
	if target := convertFingerprintToWire(ref).GetMtime(); target != nanos {
		t.Errorf("the schema carries mtime %d, want %d", target, nanos)
	}

	back := &Loopback{address: "http://127.0.0.1:1", token: "given-out"}
	want := "?size=12&mtime=1788433445123456789"
	if targetVal := back.Address(domain.Vault{ID: "01VAULT"}, ref); !strings.HasSuffix(targetVal, want) {
		t.Errorf("the address is %q, and it should end %q", targetVal, want)
	}
}

// A stamp a clock handed out carries a monotonic reading and a zone; an address
// and a message carry neither. The file an address was given out for is still
// that file when the address comes back.
func TestAFileStampedByAClockIsStillItselfThroughTheSchema(t *testing.T) {
	ref := domain.Fingerprint{Path: "recordings/Compost.mp3", Size: 3, ModTime: time.Now()}
	if back := convertFingerprintFromWire(convertFingerprintToWire(ref)); !back.IsUnchanged(ref) {
		t.Errorf("the message carried the file back as %+v, and it is %+v", back, ref)
	}
}

// The window's own fingerprint keys both caches of drawn pages. A time.Time in
// a map key compares by zone and by monotonic reading as well as by instant, so
// one picture would be two keys and the cache would never answer.
func TestTheWindowsFingerprintIsAKeyOfPlainNumbers(t *testing.T) {
	target := reflect.TypeOf(document.Fingerprint{})
	if !target.Comparable() {
		t.Fatal("the window's fingerprint cannot be a map key at all")
	}
	for item := range target.NumField() {
		if field := target.Field(item); field.Type == reflect.TypeOf(time.Time{}) {
			t.Errorf("%s is a time.Time, and this struct is a map key", field.Name)
		}
	}
}

// Nothing said about a file is not the epoch: a client presenting an empty
func TestAnEmptyStampIsNoStampAndNotTheEpoch(t *testing.T) {
	if back := convertFingerprintFromWire(convertFingerprintToWire(domain.Fingerprint{})); !back.IsZero() {
		t.Errorf("nothing said about a file came back as %+v", back)
	}
	if target := stamp(time.Time{}); target != 0 {
		t.Errorf("no stamp goes on the wire as %d", target)
	}
}
