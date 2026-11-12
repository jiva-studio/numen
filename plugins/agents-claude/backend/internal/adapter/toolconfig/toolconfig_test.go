package toolconfig

import (
	"os"
	"runtime"
	"testing"
)

func TestWriteRequiresBothUrlAndToken(t *testing.T) {
	file, err := (Files{}).Write("http://127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if file != "" {
		t.Errorf("expected no config file when token is empty, got %q", file)
	}
	file2, err := (Files{}).Write("", "token")
	if err != nil {
		t.Fatal(err)
	}
	if file2 != "" {
		t.Errorf("expected no config file when url is empty, got %q", file2)
	}
}

// The file names the server of the vault with the token, and only its owner may read it.
func TestTheFileNamesTheServerWithTheTokenForItsOwnerAlone(t *testing.T) {
	path, err := (Files{}).Write("http://127.0.0.1:4100/mcp", "orchard-token")
	if err != nil || path == "" {
		t.Fatalf("Write = %q, %v", path, err)
	}
	t.Cleanup(func() { _ = (Files{}).Remove(path) })
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
	want := `{"mcpServers":{"numen":{"type":"http","url":"http://127.0.0.1:4100/mcp","headers":{"Authorization":"Bearer orchard-token"}}}}`
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
