package agentrun

import (
	"path/filepath"
	"slices"
	"testing"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

func TestDiscoverOnPath(t *testing.T) {
	_, bin := isolateMachine(t)
	want := filepath.Join(bin, "tool")
	writeExecutable(t, want, 0o755)

	got, installed := Discover(Program{Names: []string{"tool"}})
	if !installed || got != want {
		t.Fatalf("Discover() = %q, %v; want %q, true", got, installed, want)
	}
}

func TestDiscoverInExistardPlaces(t *testing.T) {
	home, _ := isolateMachine(t)
	want := filepath.Join(home, ".local/bin/tool")
	writeExecutable(t, want, 0o755)

	got, installed := Discover(Program{Names: []string{"tool"}})
	if !installed || got != want {
		t.Fatalf("Discover() = %q, %v; want %q, true", got, installed, want)
	}
}

func TestDiscoverInCustomPlaces(t *testing.T) {
	home, _ := isolateMachine(t)
	custom := filepath.Join(home, ".custom/bin/tool")
	writeExecutable(t, custom, 0o755)

	got, installed := Discover(Program{
		Names:  []string{"tool"},
		Places: []string{"~/.custom/bin/tool"},
	})
	if !installed || got != custom {
		t.Fatalf("Discover() = %q, %v; want %q, true", got, installed, custom)
	}
}

func TestDiscoverNotFoundReturnsBareName(t *testing.T) {
	isolateMachine(t)

	got, installed := Discover(Program{
		Names:  []string{"tool", "tool-alias"},
		Places: []string{"~/.custom/bin/tool"},
	})
	if installed || got != "tool" {
		t.Fatalf("Discover() = %q, %v; want %q, false", got, installed, "tool")
	}
}

func TestDiscoverEmptyNamesNotFound(t *testing.T) {
	isolateMachine(t)

	got, installed := Discover(Program{})
	if installed || got != "" {
		t.Fatalf("Discover() = %q, %v; want %q, false", got, installed, "")
	}
}

func TestExistardPlaces(t *testing.T) {
	got := StandardPlaces("tool")
	want := []string{
		"~/.local/bin/tool",
		"/opt/homebrew/bin/tool",
		"/usr/local/bin/tool",
		"~/.nix-profile/bin/tool",
		"/run/current-system/sw/bin/tool",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("StandardPlaces() = %v, want %v", got, want)
	}
}

func TestProgramCommand(t *testing.T) {
	commands := []*numenv1.AgentProgramCommand{
		{Program: "codex", Command: []string{"/bin/codex", "exec"}},
		{Program: "other", Command: []string{"/bin/other"}},
	}
	if got := ProgramCommand(commands, "codex"); !slices.Equal(got, []string{"/bin/codex", "exec"}) {
		t.Fatalf("ProgramCommand() = %v, want %v", got, []string{"/bin/codex", "exec"})
	}
	if got := ProgramCommand(commands, "missing"); got != nil {
		t.Fatalf("ProgramCommand() = %v, want nil", got)
	}
}
