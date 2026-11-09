package appstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceReportsATargetItCannotRenameOver(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	dir := t.TempDir()
	target := filepath.Join(dir, "vaults.json")
	if err := os.MkdirAll(filepath.Join(target, "inside"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replace(target, []byte("{}")); err == nil {
		t.Fatal("a folder was replaced by a file")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %v, want only the target", entries)
	}
}
