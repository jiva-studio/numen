package editor

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// An address and a message both carry a modification time as nanoseconds since
// the epoch, and as that number and no other.
func TestTheSchemaAndTheAddressCarryNanosecondsSinceTheEpoch(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	const nanos = 1788433445123456789

	ref := domain.Fingerprint{Path: "library/Compost.pdf", Size: 12, ModTime: at}
	if held := fingerprintOf(ref).GetMtime(); held != nanos {
		t.Errorf("the schema carries mtime %d, want %d", held, nanos)
	}

	back := &Loopback{address: "http://127.0.0.1:1", token: "given-out"}
	want := "?size=12&mtime=1788433445123456789"
	if held := back.Address(domain.Vault{ID: "01VAULT"}, ref); !strings.HasSuffix(held, want) {
		t.Errorf("the address is %q, and it should end %q", held, want)
	}
}

// A stamp a clock handed out carries a monotonic reading and a zone; an address
// and a message carry neither. The file an address was given out for is still
// that file when the address comes back.
func TestAFileStampedByAClockIsStillItselfThroughTheSchema(t *testing.T) {
	ref := domain.Fingerprint{Path: "recordings/Compost.mp3", Size: 3, ModTime: time.Now()}
	if back := refOf(fingerprintOf(ref)); !back.Unchanged(ref) {
		t.Errorf("the message carried the file back as %+v, and it is %+v", back, ref)
	}
}

// The window's own fingerprint keys both caches of drawn pages. A time.Time in
// a map key compares by zone and by monotonic reading as well as by instant, so
// one picture would be two keys and the cache would never answer.
func TestTheWindowsFingerprintIsAKeyOfPlainNumbers(t *testing.T) {
	held := reflect.TypeOf(fingerprint{})
	if !held.Comparable() {
		t.Fatal("the window's fingerprint cannot be a map key at all")
	}
	for at := range held.NumField() {
		if field := held.Field(at); field.Type == reflect.TypeOf(time.Time{}) {
			t.Errorf("%s is a time.Time, and this struct is a map key", field.Name)
		}
	}
}

// Nothing said about a file is not the epoch: a client presenting an empty
// fingerprint is saying it does not know, and a write is not held to 1970.
func TestAnEmptyStampIsNoStampAndNotTheEpoch(t *testing.T) {
	if back := refOf(fingerprintOf(domain.Fingerprint{})); !back.IsZero() {
		t.Errorf("nothing said about a file came back as %+v", back)
	}
	if held := stamp(time.Time{}); held != 0 {
		t.Errorf("no stamp goes on the wire as %d", held)
	}
}
