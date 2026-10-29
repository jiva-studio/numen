package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Size and modification time are the invalidation key: a walk that hashed every
// file would read the whole vault to discover that nothing changed.
func TestAFileIsSkippedOnItsSizeAndItsTime(t *testing.T) {
	stamp := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.FileStat{Path: "a.md", Kind: domain.KindNote, Size: 10, ModTime: stamp}
	if !was.IsUnchanged(domain.FileStat{Size: 10, ModTime: stamp}) {
		t.Error("an unchanged file is read again")
	}
	if was.IsUnchanged(domain.FileStat{Size: 11, ModTime: stamp}) {
		t.Error("a file that grew is skipped")
	}
	if was.IsUnchanged(domain.FileStat{Size: 10, ModTime: stamp.Add(time.Nanosecond)}) {
		t.Error("a file written again is skipped")
	}
	// The path is not the key: a file stat the index returns carries
	// neither path nor kind.
	if !was.IsUnchanged(domain.FileStat{Path: "elsewhere.md", Size: 10, ModTime: stamp}) {
		t.Error("the path decided whether the file was read")
	}
}

// A stamp in seconds matches nothing a walk took, so a file carrying one reads
// as changed on every scan.
func TestAStampInSecondsIsNotTheSameFileAsOneInNanoseconds(t *testing.T) {
	stamp := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.FileStat{Size: 10, ModTime: stamp}
	if was.IsUnchanged(domain.FileStat{Size: 10, ModTime: stamp.Truncate(time.Second)}) {
		t.Error("a stamp in seconds passed for the file a stamp in nanoseconds describes")
	}
}

// Two values naming one instant in two zones are one file. `==` on a time.Time
// compares the zone as well, so a file stat read back somewhere with a
// different location would read as changed and the whole vault as rewritten.
func TestOneInstantInTwoZonesIsOneFile(t *testing.T) {
	stamp := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	elsewhere := stamp.In(time.FixedZone("Kathmandu", 5*3600+45*60))
	if stamp == elsewhere {
		t.Fatal("the two are the same value, and this test proves nothing")
	}
	was := domain.FileStat{Size: 10, ModTime: stamp}
	if !was.IsUnchanged(domain.FileStat{Size: 10, ModTime: elsewhere}) {
		t.Error("the same instant in another zone read as another file")
	}
}

// A stamp a clock handed out carries a monotonic reading, which `==` compares
// and no round trip preserves.
func TestAStampCarryingAMonotonicReadingIsTheSameFileWithoutIt(t *testing.T) {
	stamp := time.Now()
	if stamp.Round(0) == stamp {
		t.Skip("this clock hands out no monotonic reading")
	}
	was := domain.FileStat{Size: 10, ModTime: stamp}
	if !was.IsUnchanged(domain.FileStat{Size: 10, ModTime: time.Unix(0, stamp.UnixNano())}) {
		t.Error("a stamp read back off the wire read as another file")
	}
}

// A field named for a moment holds a moment. An integer here is a unit that is not
// stated, and any code counting anything would assign to it.
func TestTheStampIsAnInstant(t *testing.T) {
	target, found := reflect.TypeOf(domain.FileStat{}).FieldByName("ModTime")
	if !found {
		t.Fatal("a fingerprint carries no modification time")
	}
	if target.Type != reflect.TypeOf(time.Time{}) {
		t.Errorf("the stamp is %v, and a file's modification time is an instant", target.Type)
	}
}

func TestAFingerprintIsZeroOnlyWhenItNamesNothing(t *testing.T) {
	if !(domain.FileStat{}).IsZero() {
		t.Error("an empty fingerprint is not zero")
	}
	stamp := time.Date(2026, 9, 3, 11, 4, 5, 0, time.UTC)
	for name, f := range map[string]domain.FileStat{
		"path": {Path: "a.md"},
		"kind": {Kind: domain.KindNote},
		"size": {Size: 1},
		"time": {ModTime: stamp},
	} {
		if f.IsZero() {
			t.Errorf("a fingerprint with only its %s set is zero", name)
		}
	}
}
