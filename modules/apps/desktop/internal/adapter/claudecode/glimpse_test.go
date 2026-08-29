package claudecode

import (
	"slices"
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

// The window is not a Claude Code session, so the variables naming one are
// stripped from what the agent is started with.
func TestTheChildIsNotToldAboutSomebodyElsesSession(t *testing.T) {
	given := []string{
		"CLAUDECODE=1",
		"CLAUDE_CODE_SESSION_ID=abc",
		"CLAUDE_CODE_CHILD_SESSION=1",
		"CLAUDE_CODE_ENTRYPOINT=cli",
		"CLAUDE_CODE_EXECPATH=/somewhere/claude",
		"CLAUDE_CODE_MESSAGING_SOCKET=/run/user/1000/x.sock",
		"CLAUDE_CODE_MESSAGING_TOKEN=secret",
		"CLAUDE_PID=1234",
		"CLAUDE_EFFORT=high",
		"HOME=/home/somebody",
		"PATH=/usr/bin",
		"ANTHROPIC_API_KEY=sk-test",
		"LANG=ru_RU.UTF-8",
	}

	kept := map[string]bool{}
	for _, entry := range environment(given) {
		name, _, _ := strings.Cut(entry, "=")
		kept[name] = true
	}

	// Named one at a time: a check that only counts passes when the command line
	// grows a variable nobody added to the list.
	for _, gone := range []string{
		"CLAUDECODE", "CLAUDE_CODE_SESSION_ID", "CLAUDE_CODE_CHILD_SESSION",
		"CLAUDE_CODE_ENTRYPOINT", "CLAUDE_CODE_EXECPATH",
		"CLAUDE_CODE_MESSAGING_SOCKET", "CLAUDE_CODE_MESSAGING_TOKEN",
		"CLAUDE_PID", "CLAUDE_EFFORT",
	} {
		if kept[gone] {
			t.Errorf("%s was passed on", gone)
		}
	}
	// What a program needs, and what says how to reach a model, are the
	// installation's and go through.
	for _, held := range []string{"HOME", "PATH", "ANTHROPIC_API_KEY", "LANG"} {
		if !kept[held] {
			t.Errorf("%s was taken away", held)
		}
	}
}

// An entry with no name and no value is not a variable, and must not become one.
func TestAnEntryWithNoNameIsPassedOverUntouched(t *testing.T) {
	got := environment([]string{"", "=x", "HOME=/home/somebody"})
	if !slices.Contains(got, "HOME=/home/somebody") {
		t.Errorf("got %q", got)
	}
}
