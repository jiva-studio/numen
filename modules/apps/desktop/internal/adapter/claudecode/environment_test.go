package claudecode

import (
	"slices"
	"strings"
	"testing"
)

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
