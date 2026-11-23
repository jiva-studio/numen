package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTheStateFolderIsCreatedWhenItIsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "server")

	if err := prepareStateDir(dir); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("the state folder is %v, %v, want a folder", info, err)
	}
}
