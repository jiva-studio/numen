package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun/agenttest"
)

func writeProgram(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestFindReturnsTheFirstNameOnPath(t *testing.T) {
	agenttest.NeedsShell(t)
	binDir := t.TempDir()
	writeProgram(t, filepath.Join(binDir, "second"))
	t.Setenv("PATH", binDir)
	program := Program{Names: []string{"first", "second"}, Places: nil, Fallback: "first"}

	if got, want := program.Find(), filepath.Join(binDir, "second"); got != want {
		t.Errorf("Find() = %q, want %q", got, want)
	}
	if !program.IsInstalled() {
		t.Error("IsInstalled() = false, want true")
	}
}

func TestFindUsesPlacesAfterPath(t *testing.T) {
	agenttest.NeedsShell(t)
	home := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeProgram(t, filepath.Join(home, "bin", "tool"))
	program := Program{Names: []string{"tool"}, Places: []string{"~/bin/tool"}, Fallback: "tool"}

	if got, want := program.Find(), filepath.Join(home, "bin", "tool"); got != want {
		t.Errorf("Find() = %q, want %q", got, want)
	}
}

func TestFindReturnsTheFallbackWhereTheProgramIsAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	program := Program{Names: []string{"tool"}, Places: []string{"~/bin/tool"}, Fallback: "fallback"}

	if got := program.Find(); got != "fallback" {
		t.Errorf("Find() = %q, want fallback", got)
	}
	if program.IsInstalled() {
		t.Error("IsInstalled() = true, want false")
	}
}

func TestFindIgnoresACustomPathThatIsNotRunnable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	program := Program{Names: []string{"tool"}, Places: nil, Fallback: "tool"}

	if got := program.Find(filepath.Join(t.TempDir(), "absent")); got != "tool" {
		t.Errorf("Find() = %q, want tool", got)
	}
	if program.IsInstalled(filepath.Join(t.TempDir(), "absent")) {
		t.Error("IsInstalled() of an absent custom path = true, want false")
	}
}

func TestGetCommandReturnsTheCommandOfTheNamedProgram(t *testing.T) {
	commands := []*agentrun.AgentProgramCommand{
		{Program: "a", Command: []string{"a-bin"}},
		{Program: "b", Command: []string{"b-bin", "--flag"}},
	}

	if got := GetCommand(commands, "b"); len(got) != 2 || got[0] != "b-bin" {
		t.Errorf("GetCommand(b) = %v", got)
	}
	if got := GetCommand(commands, "c"); got != nil {
		t.Errorf("GetCommand(c) = %v, want nil", got)
	}
}
