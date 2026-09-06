package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The scan stores what this returns and resolution asks for it, so the two
// spellings of the rule are one function.
func TestANoteIsFoundByTheLastSegmentWithoutItsExtension(t *testing.T) {
	for path, want := range map[string]string{
		"Entropy.md":               "Entropy",
		"notes/Entropy.md":         "Entropy",
		"notes/deep/Entropy.md":    "Entropy",
		"Entropy":                  "Entropy",
		"notes/Lecture 3.notes.md": "Lecture 3.notes",
		"":                         "",
	} {
		if got := domain.Basename(path); got != want {
			t.Errorf("%q is found by %q, want %q", path, got, want)
		}
	}
}

// A leading dot is part of the name and not the start of an extension.
func TestALeadingDotIsPartOfTheName(t *testing.T) {
	for path, want := range map[string]string{
		".hidden.md":       ".hidden",
		".hidden":          ".hidden",
		"notes/.hidden.md": ".hidden",
	} {
		if got := domain.Basename(path); got != want {
			t.Errorf("%q is found by %q, want %q", path, got, want)
		}
	}
}

// Only a note's extension comes off a written name, and every other dot in it
// is part of the name. It is what the scan stores beside each link.
func TestALinkIsWrittenByItsLastSegment(t *testing.T) {
	for written, want := range map[string]string{
		"Entropy":                    "Entropy",
		"notes/Entropy":              "Entropy",
		"notes/Entropy.md":           "Entropy",
		"Seminar 1.2–1.3 — Lisbon":   "Seminar 1.2–1.3 — Lisbon",
		"notes/Seminar 1.2 — Lisbon": "Seminar 1.2 — Lisbon",
		".hidden":                    ".hidden",
		".md":                        ".md",
		"":                           "",
	} {
		if got := domain.LinkName(written); got != want {
			t.Errorf("[[%s]] is written by %q, want %q", written, got, want)
		}
	}
}

// A name is compared without regard to case, and the extension is part of what
// is compared.
func TestTheExtensionComesOffALinkHoweverItIsSpelled(t *testing.T) {
	for written, want := range map[string]string{
		"Entropy.MD":       "Entropy",
		"Entropy.Md":       "Entropy",
		"notes/Entropy.mD": "Entropy",
		".MD":              ".MD",
	} {
		if got := domain.LinkName(written); got != want {
			t.Errorf("[[%s]] is written by %q, want %q", written, got, want)
		}
	}
}

// A note can answer to two seats and is shown in one place, so it takes the
// first it qualifies for.
func TestASeatIsTakenInOneOrder(t *testing.T) {
	order := []domain.Relation{domain.SeatParent, domain.SeatChild, domain.SeatJump, domain.SeatSibling}
	for i := 1; i < len(order); i++ {
		if domain.SeatRank(order[i-1]) >= domain.SeatRank(order[i]) {
			t.Errorf("%q does not come before %q", order[i-1], order[i])
		}
	}
	if domain.SeatRank("whatever") <= domain.SeatRank(domain.SeatSibling) {
		t.Error("a seat nobody decided on comes before one that was")
	}
}

func TestTheFocusIsNotRelatedToItself(t *testing.T) {
	held := domain.Neighbourhood{Focus: domain.NoteRef{Path: "notes/Entropy.md"}}
	held.Take(domain.NoteRef{Path: "notes/Entropy.md"}, domain.Neighbour{Seat: domain.SeatParent})
	if len(held.Related) != 0 {
		t.Fatalf("the focus was seated beside itself: %v", held.Related)
	}
	held.Take(domain.NoteRef{Path: "notes/Order.md", Title: "Order"}, domain.Neighbour{Seat: domain.SeatChild})
	if len(held.Related) != 1 {
		t.Fatalf("got %v", held.Related)
	}
	if held.Related[0].Path != "notes/Order.md" || held.Related[0].Title != "Order" {
		t.Errorf("the note seated is %+v", held.Related[0])
	}
}

// Size and modification time are the invalidation key: a walk that hashed every
// file would read the whole vault to discover that nothing changed.
func TestAFileIsSkippedOnItsSizeAndItsTime(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.Fingerprint{Path: "a.md", Kind: domain.KindNote, Size: 10, ModTime: at}
	if !was.Unchanged(domain.Fingerprint{Size: 10, ModTime: at}) {
		t.Error("an unchanged file is read again")
	}
	if was.Unchanged(domain.Fingerprint{Size: 11, ModTime: at}) {
		t.Error("a file that grew is skipped")
	}
	if was.Unchanged(domain.Fingerprint{Size: 10, ModTime: at.Add(time.Nanosecond)}) {
		t.Error("a file written again is skipped")
	}
	// The path is not the key: a fingerprint the index hands back carries
	// neither path nor kind.
	if !was.Unchanged(domain.Fingerprint{Path: "elsewhere.md", Size: 10, ModTime: at}) {
		t.Error("the path decided whether the file was read")
	}
}

// A stamp in seconds matches nothing a walk took, so a file carrying one reads
// as changed on every scan.
func TestAStampInSecondsIsNotTheSameFileAsOneInNanoseconds(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.Fingerprint{Size: 10, ModTime: at}
	if was.Unchanged(domain.Fingerprint{Size: 10, ModTime: at.Truncate(time.Second)}) {
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
	if !was.Unchanged(domain.Fingerprint{Size: 10, ModTime: elsewhere}) {
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
	if !was.Unchanged(domain.Fingerprint{Size: 10, ModTime: time.Unix(0, at.UnixNano())}) {
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
