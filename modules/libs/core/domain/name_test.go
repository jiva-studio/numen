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
	was := domain.Fingerprint{Path: "a.md", Kind: domain.KindNote, Size: 10, ModTime: 100}
	if !was.Unchanged(domain.Fingerprint{Size: 10, ModTime: 100}) {
		t.Error("an unchanged file is read again")
	}
	if was.Unchanged(domain.Fingerprint{Size: 11, ModTime: 100}) {
		t.Error("a file that grew is skipped")
	}
	if was.Unchanged(domain.Fingerprint{Size: 10, ModTime: 101}) {
		t.Error("a file written again is skipped")
	}
	// The path is not the key: a fingerprint the index hands back carries
	// neither path nor kind.
	if !was.Unchanged(domain.Fingerprint{Path: "elsewhere.md", Size: 10, ModTime: 100}) {
		t.Error("the path decided whether the file was read")
	}
}

// The stamp is nanoseconds since the epoch. An adapter filling it with seconds
// leaves every file looking changed on every scan, and the number alone says
// nothing about which of the two it holds.
func TestAStampInSecondsIsNotTheSameFileAsOneInNanoseconds(t *testing.T) {
	at := time.Date(2026, 9, 3, 11, 4, 5, 123456789, time.UTC)
	was := domain.Fingerprint{Size: 10, ModTime: domain.ModTimeOf(at)}
	if was.Unchanged(domain.Fingerprint{Size: 10, ModTime: domain.ModTime(at.Unix())}) {
		t.Error("a stamp in seconds passed for the file a stamp in nanoseconds describes")
	}
	// Equal and not ==, because the instant a stamp names carries neither the
	// zone nor the monotonic reading the one it was made from may have.
	if named := was.ModTime.Time(); !named.Equal(at) {
		t.Errorf("the stamp names %v, and it was made from %v", named, at)
	}
}

// The stamp has a type of its own, so an integer of unstated unit cannot be put
// here without a conversion that says what it is being taken for.
func TestTheStampIsNotABareInteger(t *testing.T) {
	held, found := reflect.TypeOf(domain.Fingerprint{}).FieldByName("ModTime")
	if !found {
		t.Fatal("a fingerprint carries no modification time")
	}
	if held.Type != reflect.TypeOf(domain.ModTime(0)) {
		t.Errorf("the stamp is %v, which anything counting anything assigns to", held.Type)
	}
}
