package executable

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func withoutClaudeAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("CLAUDE_COMMAND", "")
	if found := (Finder{}).Find(""); found != "" {
		t.Skipf("Claude Code is installed at %s", found)
	}
}

func writeRunnableFile(t *testing.T, pos string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pos), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pos, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// The command the environment names comes first, then Claude Code on the path, then in a place it
// is installed to.
func TestFind_LookupPriority(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the places are unix paths")
	}
	withoutClaudeAnywhere(t)
	places := []string{"~/.local/bin/claude"}
	if got := findIn(places, ""); got != "" {
		t.Fatalf("found %q on an empty machine", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	inPlace := filepath.Join(home, ".local", "bin", "claude")
	writeRunnableFile(t, inPlace)
	if got := findIn(places, ""); got != inPlace {
		t.Fatalf("found %q, want %q", got, inPlace)
	}

	bin := t.TempDir()
	t.Setenv("PATH", bin)
	onPath := filepath.Join(bin, "claude")
	writeRunnableFile(t, onPath)
	if got := findIn(places, ""); got != onPath {
		t.Fatalf("found %q, want %q", got, onPath)
	}

	t.Setenv("CLAUDE_COMMAND", "/opt/claude-a")
	if got := findIn(places, ""); got != "/opt/claude-a" {
		t.Fatalf("found %q", got)
	}
	t.Setenv("NUMEN_CLAUDE_COMMAND", "/opt/claude-b")
	if got := findIn(places, ""); got != "/opt/claude-b" {
		t.Fatalf("found %q", got)
	}
}

func TestDiscoveryCustomPathAndGlobEscaping(t *testing.T) {
	if len(getSearchPlaces()) == 0 {
		t.Fatal("expected non-empty default search places")
	}

	withoutClaudeAnywhere(t)
	runnable := filepath.Join(t.TempDir(), "custom-claude")
	writeRunnableFile(t, runnable)
	t.Setenv("NUMEN_CLAUDE_COMMAND", "/opt/env-claude")

	// A runnable custom path takes precedence over environment variables.
	if got := findIn(nil, runnable); got != runnable {
		t.Errorf("findIn(runnable) = %q, want %q", got, runnable)
	}
	// An empty or non-runnable custom path falls through to the environment variable.
	if got := findIn(nil, ""); got != "/opt/env-claude" {
		t.Errorf("findIn(\"\") = %q, want /opt/env-claude", got)
	}
	if got := findIn(nil, "/nonexistent/claude"); got != "/opt/env-claude" {
		t.Errorf("findIn(nonexistent) = %q, want /opt/env-claude", got)
	}

	// An empty HOME skips ~/ places and uses absolute places.
	t.Setenv("NUMEN_CLAUDE_COMMAND", "")
	t.Setenv("HOME", "")
	if got := findInPlaces([]string{"~/.local/bin/claude", runnable}); got != runnable {
		t.Errorf("findInPlaces with empty HOME = %q, want %q", got, runnable)
	}

	// Glob characters in a path are escaped.
	escaped := escapeGlob(`/tmp/a*b?c[d\e`)
	wantEscaped := `/tmp/a[*]b[?]c[[]d\\e`
	if runtime.GOOS == "windows" {
		wantEscaped = `/tmp/a[*]b[?]c[[]d\e`
	}
	if escaped != wantEscaped {
		t.Errorf("escapeGlob = %q, want %q", escaped, wantEscaped)
	}
}
