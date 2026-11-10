package indexfolder_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/indexfolder"
)

func TestPrepareCreatesTheFolderOfTheDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "numen", "index.db")

	got, err := indexfolder.Prepare(path)

	if err != nil || got != path {
		t.Fatalf("Prepare returned %q, %v", got, err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || !info.IsDir() {
		t.Errorf("the folder of the database: %v, %v", info, err)
	}
}

func TestPrepareRefusesAnEmptyPath(t *testing.T) {
	if _, err := indexfolder.Prepare(""); !errors.Is(err, indexfolder.ErrNoPath) {
		t.Fatalf("Prepare of no path returned %v, want ErrNoPath", err)
	}
}
