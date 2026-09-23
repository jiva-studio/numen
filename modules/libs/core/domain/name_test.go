package domain_test

import (
	"testing"

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
