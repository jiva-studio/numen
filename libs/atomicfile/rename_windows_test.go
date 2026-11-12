package atomicfile

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A file is renamed over while another program has it open. Windows rejects a
// rename over a file opened without a delete share, and programs that do this
// (the search indexer, a backup reader, a virus scanner) release the file
// within moments.
func TestRenameRetriesWhileAHandleIsHeld(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	if err := os.WriteFile(target, []byte("the old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	siblingPath := filepath.Join(dir, ".state.json.new")
	if err := os.WriteFile(siblingPath, []byte("the new content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Both ends of the move are the root's own names, so neither can be carried
	// out of it by a link put in the way.
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	handle := openHandle(t, target)
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		time.Sleep(50 * time.Millisecond)
		syscall.CloseHandle(handle)
	}()

	if err := Rename(t.Context(), root, ".state.json.new", "state.json"); err != nil {
		<-closed
		t.Fatalf("the rename failed while the file was held: %v", err)
	}
	<-closed

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "the new content\n" {
		t.Errorf("the file holds %q", got)
	}
}

func openHandle(t *testing.T, path string) syscall.Handle {
	t.Helper()

	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(
		name,
		syscall.GENERIC_READ,
		syscall.FILE_SHARE_READ,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

// A file that is never released fails, and the save reports the failure.
func TestRenameOverAHandleNothingReleasesIsRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	if err := os.WriteFile(target, []byte("the old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	siblingPath := filepath.Join(dir, ".state.json.new")
	if err := os.WriteFile(siblingPath, []byte("the new content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	handle := openHandle(t, target)
	defer syscall.CloseHandle(handle)

	if err := Rename(t.Context(), root, ".state.json.new", "state.json"); err == nil {
		t.Error("the rename was reported as landed over a file nothing let go of")
	}
}
