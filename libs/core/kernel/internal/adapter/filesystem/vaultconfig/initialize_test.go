package vaultconfig

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInitializeWritesTheIdentityFileAloneAndKeepsItOnASecondCall(t *testing.T) {
	root := t.TempDir()
	first, err := Initialize(root, DefaultServiceDir, time.Unix(1, 0))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(root, DefaultServiceDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != configName {
		t.Fatalf("service folder holds %v, want only %s", entries, configName)
	}
	second, err := Initialize(root, DefaultServiceDir, time.Unix(2, 0))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("identity %s became %s", first.ID, second.ID)
	}
}
