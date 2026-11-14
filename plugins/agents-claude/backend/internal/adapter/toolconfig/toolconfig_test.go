package toolconfig

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestWriteRequiresBothUrlAndToken(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	for _, one := range [][2]string{{"http://127.0.0.1", ""}, {"", "token"}} {
		file, names, err := (Files{}).Write(one[0], one[1])
		if err != nil || file != "" || names != nil {
			t.Errorf("Write(%q, %q) = %q, %v, %v, want no file", one[0], one[1], file, names, err)
		}
	}
}

// The file names the server of the vault with the token, and only its owner may read it.
func TestTheFileNamesTheServerWithTheTokenForItsOwnerAlone(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	path, names, err := (Files{}).Write("http://127.0.0.1:4100/mcp", "orchard-token")
	if err != nil || path == "" || len(names) != 0 {
		t.Fatalf("Write = %q, %v, %v", path, names, err)
	}
	t.Cleanup(func() {
		if err := (Files{}).Remove(path); err != nil {
			t.Error(err)
		}
	})
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("the file is readable as %v", info.Mode().Perm())
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"mcpServers":{"numen":{"headers":{"Authorization":"Bearer orchard-token"},"type":"http","url":"http://127.0.0.1:4100/mcp"}}}`
	if string(raw) != want {
		t.Errorf("the file holds %s, want %s", raw, want)
	}
	if err := (Files{}).Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := (Files{}).Remove(path); err != nil {
		t.Errorf("removing a file that is gone failed: %v", err)
	}
}

// The person's own servers of Claude Code are named in the file with the server of the vault, which a
// server of the person by its name does not replace.
func TestTheFileNamesThePersonsOwnServersWithTheVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	config := `{"numStartups":3,"mcpServers":{"weather":{"type":"stdio","command":"weather-mcp"},"numen":{"type":"http","url":"http://elsewhere.test"}}}`
	if err := os.WriteFile(filepath.Join(dir, ".claude.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	path, names, err := (Files{}).Write("http://127.0.0.1:4100/mcp", "orchard-token")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := (Files{}).Remove(path); err != nil {
			t.Error(err)
		}
	})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"mcpServers":{"numen":{"headers":{"Authorization":"Bearer orchard-token"},"type":"http","url":"http://127.0.0.1:4100/mcp"},` +
		`"weather":{"type":"stdio","command":"weather-mcp"}}}`
	if string(raw) != want || !slices.Equal(names, []string{"weather"}) {
		t.Errorf("the file holds %s and names %v, want %s and the weather server", raw, names, want)
	}
}

func TestHomeFallbackAndErrorPaths(t *testing.T) {
	// Fallback to HOME when CLAUDE_CONFIG_DIR is empty.
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, ".claude.json"), []byte(`{"mcpServers":{"home-srv":{"type":"stdio"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	path, names, err := (Files{}).Write("http://127.0.0.1:4100/mcp", "token")
	if err != nil || !slices.Equal(names, []string{"home-srv"}) {
		t.Fatalf("Write with HOME fallback = (%q, %v, %v)", path, names, err)
	}
	if err := (Files{}).Remove(path); err != nil {
		t.Fatal(err)
	}

	// Empty HOME and empty CLAUDE_CONFIG_DIR does not read .claude.json from the current directory.
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, ".claude.json"), []byte(`{"mcpServers":{"cwd-srv":{"type":"stdio"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	t.Setenv("HOME", "")
	if got := readUserServers(); len(got) != 0 {
		t.Errorf("readUserServers with empty HOME = %v, want empty", got)
	}

	// Broken JSON in .claude.json after a valid entry returns no servers.
	brokenDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", brokenDir)
	if err := os.WriteFile(filepath.Join(brokenDir, ".claude.json"), []byte(`{"mcpServers":{"valid":{"type":"stdio"},"broken":bad}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readUserServers(); len(got) != 0 {
		t.Errorf("readUserServers with broken JSON = %v, want empty", got)
	}

	// Marshal failure in writeServerFile.
	if _, err := writeServerFile("numen-tools-*.json", map[string]any{"bad": math.NaN()}); err == nil {
		t.Error("expected error marshaling NaN")
	}

	// Write failure when TMPDIR does not exist.
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing-tmpdir"))
	if _, _, err := (Files{}).Write("http://127.0.0.1:4100/mcp", "token"); err == nil {
		t.Error("expected Write to fail when TMPDIR is missing")
	}
	t.Setenv("TMPDIR", t.TempDir())

	// finishTempFile cleans up the temp file when writing fails on a closed file.
	tempFile, err := os.CreateTemp(t.TempDir(), "closed-*.json")
	if err != nil {
		t.Fatal(err)
	}
	tempName := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		t.Fatal(err)
	}
	if gotPath, err := finishTempFile(tempFile, []byte("{}")); err == nil || gotPath != "" {
		t.Errorf("finishTempFile on closed file = (%q, %v), want empty path and error", gotPath, err)
	}
	if _, err := os.Stat(tempName); !os.IsNotExist(err) {
		t.Errorf("expected failed temp file %s to be removed, stat err = %v", tempName, err)
	}

	// Remove("") is a no-op; Remove on a non-empty directory returns an error.
	if err := (Files{}).Remove(""); err != nil {
		t.Errorf("Remove(\"\") = %v, want nil", err)
	}
	nonEmptyDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Files{}).Remove(nonEmptyDir); err == nil {
		t.Error("expected Remove on non-empty directory to fail")
	}
}
