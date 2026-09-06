package claudecode

import (
	"strings"
	"testing"
)

// Arguments arrive in pieces the model chose, not pieces that are valid JSON or
// even whole characters. Every cut through a Cyrillic body is a cut through the
// middle of a rune.
func TestGlimpsedSurvivesEveryCut(t *testing.T) {
	whole := `{"title":"Сарай и \"ключ\"","body":"Ключ от сарая лежит под кирпичом у двери."}`
	for at := 0; at <= len(whole); at++ {
		got := glimpsed(whole[:at], "title")
		if got != "" && !hasPrefix(`Сарай и "ключ"`, got) {
			t.Fatalf("cut at %d read %q", at, got)
		}
	}
	if got := glimpsed(whole, "title"); got != `Сарай и "ключ"` {
		t.Errorf("whole reads %q", got)
	}
}

func TestGlimpsedFindsNothingItWasNotGiven(t *testing.T) {
	for _, arguments := range []string{
		``, `{`, `{"title`, `{"title"`, `{"title":`, `{"title":"`,
		`{"other":"x"}`, `{"titles":"x"}`, `{"title":5}`, `{"title":null}`,
		`{"title":"x\`,
	} {
		if got := glimpsed(arguments, "title"); got != "" && got != "x" {
			t.Errorf("%q read %q", arguments, got)
		}
	}
	if got := glimpsed(`{"title":"x"}`, ""); got != "" {
		t.Errorf("no field read %q", got)
	}
}

// An escape that names a character by number arrives in six pieces, and five of
// them are not a character.
func TestGlimpsedSurvivesAHalfWrittenEscape(t *testing.T) {
	whole := `{"title":"a\u043cb"}`
	for at := 0; at <= len(whole); at++ {
		got := glimpsed(whole[:at], "title")
		if got != "" && !hasPrefix("a\u043cb", got) && !hasPrefix("a", got) {
			t.Fatalf("cut at %d read %q", at, got)
		}
		if strings.ContainsRune(got, '\\') {
			t.Fatalf("cut at %d read an escape as text: %q", at, got)
		}
	}
	if got := glimpsed(whole, "title"); got != "a\u043cb" {
		t.Errorf("whole reads %q", got)
	}
}

// A field whose name ends another one's is not that field.
func TestGlimpsedDoesNotMistakeOneFieldForAnother(t *testing.T) {
	if got := glimpsed(`{"subtitle":"wrong","title":"right"}`, "title"); got != "right" {
		t.Errorf("read %q", got)
	}
}

func hasPrefix(whole, part string) bool {
	return len(part) <= len(whole) && whole[:len(part)] == part
}
