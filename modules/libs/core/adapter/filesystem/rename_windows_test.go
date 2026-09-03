package filesystem

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// A note is saved while another program holds the file open. Windows refuses a
// move over a file opened without a share on deletion, and the programs that do
// it — the search indexer, a backup reader, a virus scanner — let go within
// moments.
func TestASaveWaitsOutAHandleAnotherProgramHolds(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Entropy.md")
	if err := os.WriteFile(target, []byte("what stood there\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beside := filepath.Join(dir, ".Entropy.md.new")
	if err := os.WriteFile(beside, []byte("what is saved\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handle := opened(t, target)
	closed := make(chan struct{})
	go func() {
		defer close(closed)
		time.Sleep(50 * time.Millisecond)
		windows.CloseHandle(handle)
	}()

	if err := rename(beside, target); err != nil {
		<-closed
		t.Fatalf("the save was refused while the file was held: %v", err)
	}
	<-closed

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "what is saved\n" {
		t.Errorf("the note holds %q", got)
	}
}

// A handle held on a file that nothing may move over: Windows refuses a rename
// over a file opened without FILE_SHARE_DELETE.
func opened(t *testing.T, path string) windows.Handle {
	t.Helper()

	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

// A file nothing lets go of is refused, and the save says so rather than
// waiting for as long as the program runs.
func TestASaveOverAHandleNothingLetsGoIsRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Entropy.md")
	if err := os.WriteFile(target, []byte("what stood there\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beside := filepath.Join(dir, ".Entropy.md.new")
	if err := os.WriteFile(beside, []byte("what is saved\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handle := opened(t, target)
	defer windows.CloseHandle(handle)

	if err := rename(beside, target); err == nil {
		t.Error("the save was reported as landed over a file nothing let go of")
	}
}
