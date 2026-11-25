package vaultfolder_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultfolder"
)

func writeFile(t *testing.T, path string, modified time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modified, modified); err != nil {
		t.Fatal(err)
	}
}

func TestTheStateOfAPathTellsAbsentFileEmptyAndFilledFolder(t *testing.T) {
	root := t.TempDir()
	filled := filepath.Join(root, "filled")
	if err := os.Mkdir(filled, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(filled, "a.md"), time.Now())
	file := filepath.Join(root, "file.md")
	writeFile(t, file, time.Now())
	var folders vaultfolder.Folders

	absent, absentErr := folders.GetFolderState(filepath.Join(root, "none"))
	asFile, fileErr := folders.GetFolderState(file)
	top, topErr := folders.GetFolderState(root + string(os.PathSeparator) + ".")
	asFilled, filledErr := folders.GetFolderState(filled)

	for _, err := range []error{absentErr, fileErr, topErr, filledErr} {
		if err != nil {
			t.Fatal(err)
		}
	}
	if absent.Exists {
		t.Errorf("an absent path exists: %+v", absent)
	}
	if !asFile.Exists || asFile.IsFolder {
		t.Errorf("a file is %+v", asFile)
	}
	if !top.Exists || !top.IsFolder || top.IsEmpty {
		t.Errorf("a folder holding two entries is %+v", top)
	}
	if !asFilled.IsFolder || asFilled.IsEmpty {
		t.Errorf("a filled folder is %+v", asFilled)
	}
}

func TestAFolderIsMadeEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Notes")
	var folders vaultfolder.Folders

	if err := folders.CreateFolder(path); err != nil {
		t.Fatal(err)
	}

	state, err := folders.GetFolderState(path)
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsFolder || !state.IsEmpty {
		t.Errorf("the made folder is %+v", state)
	}
}

func TestOnlyWhatWasMadeSinceTheCallIsRemoved(t *testing.T) {
	path := t.TempDir()
	since := time.Now()
	service := filepath.Join(path, ".numen")
	if err := os.Mkdir(service, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(service, "vault.json"), since)
	var folders vaultfolder.Folders

	if err := folders.RemoveCreated(path, since); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("%d entries made by the call are left", len(entries))
	}
}

func TestAFolderHoldingAFileOlderThanTheCallKeepsEverything(t *testing.T) {
	path := t.TempDir()
	since := time.Now()
	writeFile(t, filepath.Join(path, "mine.md"), since.Add(-time.Hour))
	writeFile(t, filepath.Join(path, "ours.md"), since)
	var folders vaultfolder.Folders

	if err := folders.RemoveCreated(path, since); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("%d entries are left, want both", len(entries))
	}
}

func TestAFolderIsRemovedOnlyWhenItIsEmpty(t *testing.T) {
	root := t.TempDir()
	empty := filepath.Join(root, "empty")
	filled := filepath.Join(root, "filled")
	for _, path := range []string{empty, filled} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(filled, "a.md"), time.Now())
	var folders vaultfolder.Folders

	if err := folders.RemoveEmptyFolder(empty); err != nil {
		t.Fatal(err)
	}
	if err := folders.RemoveEmptyFolder(filled); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(empty); !os.IsNotExist(err) {
		t.Errorf("the empty folder was kept: %v", err)
	}
	if _, err := os.Stat(filled); err != nil {
		t.Errorf("the filled folder was removed: %v", err)
	}
}
