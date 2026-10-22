package programs

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

func writeExecutable(t *testing.T, targetPath string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetPath, mode); err != nil {
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

func TestLocateConsumesAnyExecuteBit(t *testing.T) {
	for _, mode := range []os.FileMode{0o100, 0o010, 0o001} {
		home, _ := isolateMachine(t)
		targetPath := filepath.Join(home, "tool")
		writeExecutable(t, targetPath, mode)
		if got := Locate(nil, []string{targetPath}); got != targetPath {
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
	want := filepath.Join(t.TempDir(), "tool")
	writeExecutable(t, want, 0o755)
	if got := Locate(nil, []string{"~/.local/bin/[", want}); got != want {
		t.Fatalf("found %q, want %q", got, want)
	}
}

// The home is read as a folder name, and only the place after "~/" is a pattern.
func TestLocateReadsTheHomeLiterally(t *testing.T) {
	isolateMachine(t)
	for _, name := range []string{"[x]", "*", "?", "[", `a\b`} {
		home := filepath.Join(t.TempDir(), name)
		t.Setenv("HOME", home)
		want := filepath.Join(home, ".nvm/versions/node/v1/bin/tool")
		writeExecutable(t, want, 0o755)
		if got := Locate(nil, []string{"~/.nvm/versions/node/*/bin/tool"}); got != want {
			t.Errorf("home %q: found %q, want %q", name, got, want)
		}
	}
}

func TestEscapeGlobMatchesOnlyThePath(t *testing.T) {
	isolateMachine(t)
	for _, path := range []string{"/home/[x]", "/home/*", "/home/?", "/home/[", `/home/a\b`, "/home/plain"} {
		if ok, err := filepath.Match(escapeGlob(path), path); err != nil || !ok {
			t.Errorf("escapeGlob(%q) = %q does not match the path: %v", path, escapeGlob(path), err)
		}
	}
	for path, other := range map[string]string{"/home/[x]": "/home/x", "/home/*": "/home/a", "/home/?": "/home/a"} {
		if ok, _ := filepath.Match(escapeGlob(path), other); ok {
			t.Errorf("escapeGlob(%q) matches %q", path, other)
		}
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
func TestLocateSkipsAFileBelowAnotherCase(t *testing.T) {
	home, _ := isolateMachine(t)
	writeExecutable(t, filepath.Join(home, "Applications/Tool.app/Contents/MacOS/Tool"), 0o755)
	if got := Locate(nil, []string{"~/Applications/Tool.app/Contents/MacOS/tool"}); got != "" {
		t.Fatalf("found %q", got)
	}
}
