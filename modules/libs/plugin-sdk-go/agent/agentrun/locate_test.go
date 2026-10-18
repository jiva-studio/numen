package agentrun

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolateMachine gives the test an empty home and an empty PATH folder, and returns both.
func isolateMachine(t *testing.T) (home, bin string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the places are unix paths")
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

func TestLocateFindsNothingOnAnEmptyMachine(t *testing.T) {
	isolateMachine(t)
	if got := Locate([]string{"tool"}, []string{"~/.local/bin/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
}

func TestLocateFindsEveryNameOnThePath(t *testing.T) {
	for _, name := range []string{"tool", "tool-alias"} {
		t.Run(name, func(t *testing.T) {
			_, bin := isolateMachine(t)
			want := filepath.Join(bin, name)
			writeExecutable(t, want, 0o755)
			if got := Locate([]string{"tool", "tool-alias"}, nil); got != want {
				t.Fatalf("found %q, want %q", got, want)
			}
		})
	}
}

func TestLocateAsksThePathBeforeThePlaces(t *testing.T) {
	home, bin := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, ".local/bin/tool"), 0o755)
	want := filepath.Join(bin, "tool")
	writeExecutable(t, want, 0o755)
	if got := Locate([]string{"tool"}, []string{"~/.local/bin/tool"}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

func TestLocateTriesEveryPlace(t *testing.T) {
	home, _ := isolateMachine(t)
	want := filepath.Join(home, ".nix-profile/bin/tool")
	writeExecutable(t, want, 0o755)
	if got := Locate(nil, []string{"~/.local/bin/tool", "~/.nix-profile/bin/tool"}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

func TestLocateSkipsAFileWithoutAnExecuteBit(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, ".local/bin/tool"), 0o644)
	if got := Locate(nil, []string{"~/.local/bin/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
}

func TestLocateTakesAnyExecuteBit(t *testing.T) {
	for _, mode := range []os.FileMode{0o100, 0o010, 0o001} {
		home, _ := isolateMachine(t)
		at := filepath.Join(home, "tool")
		writeExecutable(t, at, mode)
		if got := Locate(nil, []string{at}); got != at {
			t.Errorf("mode %o: found %q", mode, got)
		}
	}
}

func TestLocateFindsTheRunnableMatchOfAPattern(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, ".nvm/versions/node/v1/bin/tool"), 0o644)
	want := filepath.Join(home, ".nvm/versions/node/v2/bin/tool")
	writeExecutable(t, want, 0o755)
	if got := Locate(nil, []string{"~/.nvm/versions/node/*/bin/tool"}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

func TestLocateSkipsADirectory(t *testing.T) {
	home, _ := isolateMachine(t)
	if err := os.MkdirAll(filepath.Join(home, ".local/bin/tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := Locate(nil, []string{"~/.local/bin/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
}

func TestLocateSkipsAPlaceThatIsNotAPattern(t *testing.T) {
	isolateMachine(t)
	t.Setenv("HOME", filepath.Join(t.TempDir(), "["))
	want := filepath.Join(t.TempDir(), "tool")
	writeExecutable(t, want, 0o755)
	if got := Locate(nil, []string{"~/.local/bin/tool", want}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

// With no home, a place under the home is skipped and is never read from the current folder; the
// places outside the home are still searched.
func TestLocateWithNoHomeSearchesOnlyThePlacesOutsideIt(t *testing.T) {
	isolateMachine(t)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
	here := t.TempDir()
	writeExecutable(t, filepath.Join(here, ".local/bin/tool"), 0o755)
	t.Chdir(here)
	if got := Locate(nil, []string{"~/.local/bin/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
	want := filepath.Join(t.TempDir(), "bin", "tool")
	writeExecutable(t, want, 0o755)
	if got := Locate(nil, []string{"~/.local/bin/tool", want}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

// A case-insensitive filesystem answers to a name in any case, and the program is the file held
// under the exact name.
func TestLocateSkipsAFileUnderAnotherCase(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, "Applications/Tool.app/Contents/MacOS/Tool"), 0o755)
	if got := Locate(nil, []string{"~/Applications/Tool.app/Contents/MacOS/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
}
