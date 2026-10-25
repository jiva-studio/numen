package agentrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTempDirCreatesAndCleansUp(t *testing.T) {
	dir, cleanup, err := CreateTempDir("numen-test-*")
	if err != nil {
		t.Fatalf("CreateTempDir() failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("expected directory at %s, err: %v", dir, err)
	}

	testFile := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}

	cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected directory to be deleted, err: %v", err)
	}

	// Calling cleanup again does not error or panic.
	cleanup()
}
