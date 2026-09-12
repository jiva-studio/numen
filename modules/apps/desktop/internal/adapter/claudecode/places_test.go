package claudecode

import (
	"os"
	"path/filepath"
	"testing"
)

// An application opened from a desktop is given the system path alone, so the
// command line is looked for where its installers put it.
func TestTheCommandLineIsFoundWhereAnInstallerPutIt(t *testing.T) {
	home := t.TempDir()
	program(t, filepath.Join(home, ".local", "bin", "claude"))

	want := filepath.Join(home, ".local", "bin", "claude")
	if got := found(home, places); got != want {
		t.Errorf("found %q, want %q", got, want)
	}
}

// A mac carries programs inside application bundles, and the command line
// found there is a name standing for the file it points at.
func TestTheCommandLineIsFoundInsideAnApplicationBundle(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(home, ".local", "share", "claude", "versions", "2.1.241")
	program(t, real)

	bundle := filepath.Join(home, "Applications", "Claude Code URL Handler.app", "Contents", "MacOS")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(bundle, "claude")
	if err := os.Symlink(real, want); err != nil {
		t.Fatal(err)
	}

	if got := found(home, places); got != want {
		t.Errorf("found %q, want %q", got, want)
	}
}

// A version folder is written by the installer, so the place names it with a
// star and any one of them answers.
func TestAVersionFolderIsExpanded(t *testing.T) {
	home := t.TempDir()
	want := filepath.Join(home, ".nvm", "versions", "node", "v20.0.0", "bin", "claude")
	program(t, want)

	if got := found(home, places); got != want {
		t.Errorf("found %q, want %q", got, want)
	}
}

// A folder of that name and a file nobody may run are both not the command
// line.
func TestWhatCannotBeRunIsNotTheCommandLine(t *testing.T) {
	onlyPlaces(t, "~/.local/bin/claude", "~/.bun/bin/claude")
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".local", "bin", "claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	unreadable := filepath.Join(home, ".bun", "bin", "claude")
	if err := os.MkdirAll(filepath.Dir(unreadable), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unreadable, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := found(home, places); got != "" {
		t.Errorf("found %q, want nothing", got)
	}
}

// A mac filesystem answers to a name in any case, and the command line is the
// file held under that name.
func TestAProgramUnderAnotherCaseIsNotTheCommandLine(t *testing.T) {
	onlyPlaces(t, "~/Applications/Claude.app/Contents/MacOS/claude")
	home := t.TempDir()
	program(t, filepath.Join(home, "Applications", "Claude.app", "Contents", "MacOS", "Claude"))

	if got := found(home, places); got != "" {
		t.Errorf("found %q, want nothing", got)
	}
}

// Nothing is looked up under a home that is not known.
func TestNoHomeMeansNoHomePlaces(t *testing.T) {
	if got := found("", []string{"~/.local/bin/claude"}); got != "" {
		t.Errorf("found %q, want nothing", got)
	}
}

// The path is what the person's own installation says, so it is asked before
// the places are.
func TestThePathIsAskedFirst(t *testing.T) {
	onPath := t.TempDir()
	program(t, filepath.Join(onPath, "claude"))
	home := t.TempDir()
	program(t, filepath.Join(home, ".local", "bin", "claude"))

	t.Setenv("PATH", onPath)
	t.Setenv("HOME", home)

	want := filepath.Join(onPath, "claude")
	if got := findCommand(); got != want {
		t.Errorf("start %q, want %q", got, want)
	}
}

// A machine with none of it starts the bare name, and that says it is not
// installed.
func TestNowhereStartsTheBareName(t *testing.T) {
	onlyPlaces(t, "~/.local/bin/claude")
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	if got := findCommand(); got != "claude" {
		t.Errorf("start %q, want %q", got, "claude")
	}
}

// onlyPlaces holds the places to the ones a test writes into, so that a machine
// running the tests with the command line installed on it answers the same.
func onlyPlaces(t *testing.T, only ...string) {
	t.Helper()
	held := places
	places = only
	t.Cleanup(func() { places = held })
}

func program(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
