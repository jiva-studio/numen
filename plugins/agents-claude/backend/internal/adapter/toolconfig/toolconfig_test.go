package toolconfig

import (
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
