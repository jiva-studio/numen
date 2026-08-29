package domain_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

func TestATitleThatIsAlreadyAFilenameIsFiledUnderItself(t *testing.T) {
	for _, title := range []string{
		"Entropy",
		"Zoë Brontë Mårtensson",
		"Холм у реки",
		"Lecture 3 — entropy, and what it is not",
	} {
		name, exact := domain.Filename(title)
		if name != title || !exact {
			t.Errorf("%q was filed as %q, exact %v", title, name, exact)
		}
	}
}

func TestACharacterAFilesystemReservesBecomesADash(t *testing.T) {
	// A slash would file the note in a folder, and the rest are reserved
	// somewhere that matters.
	name, exact := domain.Filename(`a/b\c:d*e?f"g<h>i|j`)
	if name != "a-b-c-d-e-f-g-h-i-j" {
		t.Errorf("got %q", name)
	}
	if exact {
		t.Error("the title survived the trip")
	}
}

func TestATitleIsFiledUnderNoLeadingDot(t *testing.T) {
	// A leading dot files the note where nothing looks.
	name, exact := domain.Filename(".hidden")
	if name != "hidden" || exact {
		t.Errorf("got %q, exact %v", name, exact)
	}
}

func TestATrailingDotOrSpaceIsDropped(t *testing.T) {
	for _, title := range []string{"Entropy.", "Entropy ", "Entropy. ."} {
		name, exact := domain.Filename(title)
		if name != "Entropy" {
			t.Errorf("%q was filed as %q", title, name)
		}
		if exact != (title == "Entropy ") {
			t.Errorf("%q: exact %v", title, exact)
		}
	}
}

func TestAControlCharacterIsNotInTheName(t *testing.T) {
	name, exact := domain.Filename("En\x00tro\npy")
	if name != "Entropy" || exact {
		t.Errorf("got %q, exact %v", name, exact)
	}
}

func TestATitleTooLongIsCutBetweenCharacters(t *testing.T) {
	// Most filesystems stop at 255 bytes for one component, and the cut lands
	// between characters: half a character is not a character.
	long := strings.Repeat("ṛ", 200)
	name, exact := domain.Filename(long)
	if exact {
		t.Error("a title that was cut says it survived")
	}
	if len(name) > 120 {
		t.Errorf("%d bytes", len(name))
	}
	if !utf8.ValidString(name) {
		t.Errorf("a character was cut in half: %q", name)
	}
}

func TestATitleThatCannotBeAFilenameIsNoFilename(t *testing.T) {
	for _, title := range []string{"", "   ", "...", "\x00", ". ."} {
		if name, exact := domain.Filename(title); name != "" || exact {
			t.Errorf("%q was filed as %q, exact %v", title, name, exact)
		}
	}
}

// A note is reached by a link written by its name, so every filename a title
// reduces to is one a link can be written by.
func TestEveryFilenameATitleReducesToCanBeWrittenAsALink(t *testing.T) {
	for _, title := range []string{
		"C#", "F# and C#", "#tag", "###",
		"Notes [[draft]]", "[[", "]]", "[[[deep]]]", "a [1980] note",
		"Either|Or", "TCP/IP", "note://01J8", "a:b", "a\\b",
		"  Entropy  ", " Entropy ", ".hidden", "trailing.",
		"a .", "x" + strings.Repeat("é", 200) + "#",
		"ordinary", "Ṛtu and the seasons",
	} {
		name, _ := domain.Filename(title)
		if name == "" {
			continue
		}
		if !domain.Nameable(name) {
			t.Errorf("%q is filed as %q, and no link can be written by that", title, name)
		}
		if domain.Basename(name+".md") != name {
			t.Errorf("%q is filed as %q, which is not the name a link resolves by", title, name)
		}
	}
}

// The characters a name cannot carry are reduced, and the ones it can are left
// where the person put them.
func TestAFilenameKeepsWhatALinkCanBeWrittenWith(t *testing.T) {
	for title, want := range map[string]string{
		"C#":              "C-",
		"Issue #42":       "Issue -42",
		"Notes [[draft]]": "Notes [draft]",
		"a [1980] note":   "a [1980] note",
		"Either|Or":       "Either-Or",
		"TCP/IP":          "TCP-IP",
		"Ṛtu":             "Ṛtu",
	} {
		if name, _ := domain.Filename(title); name != want {
			t.Errorf("%q is filed as %q, want %q", title, name, want)
		}
	}
}
