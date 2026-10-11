package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolateMachine gives the test an empty home, an empty PATH folder and the
// home's name in the environment, and returns both folders.
func isolateMachine(t *testing.T) (home, bin string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("search places are unix paths")
	}
	home, bin = t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin)
	return home, bin
}

func writeExecutable(t *testing.T, at string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(at, mode); err != nil {
		t.Fatal(err)
	}
}

// skipIfInstalledAtAnAbsolutePlace skips a test that needs the machine to hold
// none of the tool's fixed system locations.
func skipIfInstalledAtAnAbsolutePlace(t *testing.T, places []string) {
	t.Helper()
	for _, place := range places {
		if filepath.IsAbs(place) {
			if matches, _ := filepath.Glob(place); len(matches) > 0 {
				t.Skipf("%s exists on this machine", matches[0])
			}
		}
	}
}

type tool struct {
	name      string
	isInstall func() bool
	places    []string
	commands  []string
	homePlace string
}

func tools() []tool {
	return []tool{
		{"claude", IsClaudeInstalled, claudeSearchPlaces, []string{"claude"}, ".local/bin/claude"},
		{"antigravity", IsAntigravityInstalled, antigravitySearchPlaces, []string{"agy", "antigravity"}, ".local/bin/agy"},
		{"codex", IsCodexInstalled, codexSearchPlaces, []string{"codex"}, ".codex/bin/codex"},
	}
}

// A machine holding none of a tool has none of it.
func TestNoToolIsInstalledOnAnEmptyMachine(t *testing.T) {
	for _, tl := range tools() {
		t.Run(tl.name, func(t *testing.T) {
			isolateMachine(t)
			skipIfInstalledAtAnAbsolutePlace(t, tl.places)
			if tl.isInstall() {
				t.Fatal("reported installed")
			}
		})
	}
}

// A tool on PATH is installed, under each name it answers to.
func TestToolOnPathIsInstalled(t *testing.T) {
	for _, tl := range tools() {
		for _, command := range tl.commands {
			t.Run(command, func(t *testing.T) {
				_, bin := isolateMachine(t)
				writeExecutable(t, filepath.Join(bin, command), 0o755)
				if !tl.isInstall() {
					t.Fatal("reported missing")
				}
			})
		}
	}
}

// A tool in one of its known folders under the home is installed.
func TestToolInAHomePlaceIsInstalled(t *testing.T) {
	for _, tl := range tools() {
		t.Run(tl.name, func(t *testing.T) {
			home, _ := isolateMachine(t)
			writeExecutable(t, filepath.Join(home, tl.homePlace), 0o755)
			if !tl.isInstall() {
				t.Fatal("reported missing")
			}
		})
	}
}

// Every search place is tried, not only the first.
func TestToolInTheLastHomePlaceIsInstalled(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, ".nix-profile/bin/claude"), 0o755)
	if got := findInSearchPlaces(claudeSearchPlaces); got != filepath.Join(home, ".nix-profile/bin/claude") {
		t.Fatalf("found %q", got)
	}
}

// A file that is not executable is not the tool.
func TestFileWithoutExecutePermissionIsNotAnInstall(t *testing.T) {
	home, _ := isolateMachine(t)
	skipIfInstalledAtAnAbsolutePlace(t, claudeSearchPlaces)
	writeExecutable(t, filepath.Join(home, ".local/bin/claude"), 0o644)
	if IsClaudeInstalled() {
		t.Fatal("reported installed")
	}
}

// A wildcard place matches every folder, and the executable among them is
// found after a match that is not.
func TestWildcardPlaceFindsTheExecutableAfterAMatchThatIsNot(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, ".nvm/versions/node/v1/bin/claude"), 0o644)
	want := filepath.Join(home, ".nvm/versions/node/v2/bin/claude")
	writeExecutable(t, want, 0o755)
	if got := findInSearchPlaces([]string{"~/.nvm/versions/node/*/bin/claude"}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

// A directory is not the tool.
func TestDirectoryNamedLikeTheToolIsNotAnInstall(t *testing.T) {
	home, _ := isolateMachine(t)
	if err := os.MkdirAll(filepath.Join(home, ".local/bin/claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findInSearchPlaces([]string{"~/.local/bin/claude"}); got != "" {
		t.Fatalf("found %q", got)
	}
}

// With no home, a place under the home is skipped, not read from the current
// folder.
func TestHomePlacesAreSkippedWhenTheMachineHasNoHome(t *testing.T) {
	isolateMachine(t)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	skipIfInstalledAtAnAbsolutePlace(t, claudeSearchPlaces)
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
	here := t.TempDir()
	writeExecutable(t, filepath.Join(here, ".local/bin/claude"), 0o755)
	t.Chdir(here)
	if got := findInSearchPlaces([]string{"~/.local/bin/claude"}); got != "" {
		t.Fatalf("found %q", got)
	}
}

// Places that are not under the home are still searched when the machine has
// no home.
func TestAbsolutePlacesAreSearchedWhenTheMachineHasNoHome(t *testing.T) {
	isolateMachine(t)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
	want := filepath.Join(t.TempDir(), "bin", "claude")
	writeExecutable(t, want, 0o755)
	if got := findInSearchPlaces([]string{"~/.local/bin/claude", want}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

// A file is executable when any of its owner, group or other execute bits is
// set.
func TestAnyExecuteBitMakesAFileExecutable(t *testing.T) {
	for _, mode := range []os.FileMode{0o100, 0o010, 0o001} {
		home, _ := isolateMachine(t)
		at := filepath.Join(home, "tool")
		writeExecutable(t, at, mode)
		if got := findInSearchPlaces([]string{at}); got != at {
			t.Errorf("mode %o: found %q", mode, got)
		}
	}
}

// A place that is not a valid pattern is skipped, and the places after it are
// still searched.
func TestSearchContinuesPastAPlaceThatIsNotAPattern(t *testing.T) {
	isolateMachine(t)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "["))
	want := filepath.Join(t.TempDir(), "claude")
	writeExecutable(t, want, 0o755)
	if got := findInSearchPlaces([]string{"~/.local/bin/claude", want}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}
