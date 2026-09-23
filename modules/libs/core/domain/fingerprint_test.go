package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Size and modification time are the invalidation key: a walk that hashed every
// file would read the whole vault to discover that nothing changed.
func TestAFileIsSkippedOnItsSizeAndItsTime(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.Fingerprint{Path: "a.md", Kind: domain.KindNote, Size: 10, ModTime: at}
	if !was.IsUnchanged(domain.Fingerprint{Size: 10, ModTime: at}) {
		t.Error("an unchanged file is read again")
	}
	if was.IsUnchanged(domain.Fingerprint{Size: 11, ModTime: at}) {
		t.Error("a file that grew is skipped")
	}
	if was.IsUnchanged(domain.Fingerprint{Size: 10, ModTime: at.Add(time.Nanosecond)}) {
		t.Error("a file written again is skipped")
	}
	// The path is not the key: a fingerprint the index hands back carries
	// neither path nor kind.
	if !was.IsUnchanged(domain.Fingerprint{Path: "elsewhere.md", Size: 10, ModTime: at}) {
		t.Error("the path decided whether the file was read")
	}
}

// A stamp in seconds matches nothing a walk took, so a file carrying one reads
// as changed on every scan.
func TestAStampInSecondsIsNotTheSameFileAsOneInNanoseconds(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.Fingerprint{Size: 10, ModTime: at}
	if was.IsUnchanged(domain.Fingerprint{Size: 10, ModTime: at.Truncate(time.Second)}) {
		t.Error("a stamp in seconds passed for the file a stamp in nanoseconds describes")
	}
}

// Two values naming one instant in two zones are one file. `==` on a time.Time
// compares the zone as well, so a fingerprint read back somewhere with a
// different location would read as changed and the whole vault as rewritten.
func TestOneInstantInTwoZonesIsOneFile(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	elsewhere := at.In(time.FixedZone("Kathmandu", 5*3600+45*60))
	//nolint:gocritic // the operator is what this test is about
	if at == elsewhere {
		t.Fatal("the two are the same value, and this test proves nothing")
	}
	was := domain.Fingerprint{Size: 10, ModTime: at}
	if !was.IsUnchanged(domain.Fingerprint{Size: 10, ModTime: elsewhere}) {
		t.Error("the same instant in another zone read as another file")
	}
}

// A stamp a clock handed out carries a monotonic reading, which `==` compares
// and no round trip preserves.
func TestAStampCarryingAMonotonicReadingIsTheSameFileWithoutIt(t *testing.T) {
	at := time.Now()
	//nolint:gocritic // the operator is what this test is about
	if at.Round(0) == at {
		t.Skip("this clock hands out no monotonic reading")
	}
	was := domain.Fingerprint{Size: 10, ModTime: at}
	if !was.IsUnchanged(domain.Fingerprint{Size: 10, ModTime: time.Unix(0, at.UnixNano())}) {
		t.Error("a stamp read back off the wire read as another file")
	}
}

// A field named for a moment holds a moment. An integer here is a unit nobody
// stated, and anything counting anything assigns to it.
func TestTheStampIsAnInstant(t *testing.T) {
	held, found := reflect.TypeOf(domain.Fingerprint{}).FieldByName("ModTime")
	if !found {
		t.Fatal("a fingerprint carries no modification time")
	}
	if held.Type != reflect.TypeOf(time.Time{}) {
		t.Errorf("the stamp is %v, and a file's modification time is an instant", held.Type)
	}
}
