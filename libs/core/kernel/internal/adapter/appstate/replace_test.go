package appstate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReportsATargetItCannotRenameOver(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "01HVAULT.json")
	if err := os.MkdirAll(filepath.Join(target, "inside"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := VaultFilesAt(dir).Write(context.Background(), "01HVAULT", []byte("{}")); err == nil {
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
