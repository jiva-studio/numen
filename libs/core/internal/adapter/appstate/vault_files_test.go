package appstate_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// What was worked out is kept by the vault it was worked out for, and comes
// back as it was written.
func TestReadReturnsWhatWriteStoredForTheVault(t *testing.T) {
	files := appstate.VaultFilesAt(filepath.Join(t.TempDir(), "cache"))
	ctx := t.Context()

	if err := files.Write(ctx, "01J8F3K2M9QRSTVWXYZ012", []byte(`{"v":1}`)); err != nil {
		t.Fatal(err)
	}
	got, err := files.Read(ctx, "01J8F3K2M9QRSTVWXYZ012")
	if err != nil || string(got) != `{"v":1}` {
		t.Errorf("read back %q, %v", got, err)
	}

	if _, err := files.Read(ctx, "01J8F3K2M9QRSTVWXYZ999"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a vault nothing was kept for answers %v", err)
	}
}

// The identity is written into a file's name, so a name that is not one is
func TestAnIdentityThatIsNotOneNamesNoFile(t *testing.T) {
	files := appstate.VaultFilesAt(filepath.Join(t.TempDir(), "cache"))
	for _, id := range []domain.VaultID{"", "../numen.json", "a/b", `a\b`, "one.json"} {
		if err := files.Write(t.Context(), id, []byte("{}")); err == nil {
			t.Errorf("wrote under %q", id)
		}
	}
}

// A vault whose identity is not one is read as an error.
func TestAnIdentityThatIsNotOneReadsNothing(t *testing.T) {
	files := appstate.VaultFilesAt(t.TempDir())
	for _, id := range []domain.VaultID{"", "a/b", "one.json"} {
		if _, err := files.Read(t.Context(), id); err == nil {
			t.Errorf("read under %q", id)
		}
	}
}

// The folder is made when it is missing, and a folder that cannot be made is an
// error.
func TestAFolderThatCannotBeMadeFailsTheWrite(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := appstate.VaultFilesAt(filepath.Join(file, "cache"))
	if err := files.Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Error("wrote under a folder that is a file")
	}
}

// A file is written readable by its owner alone.
func TestASavedFileIsTheOwnersAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows files have no permission bits to read")
	}
	dir := filepath.Join(t.TempDir(), "cache")
	if err := appstate.VaultFilesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "vault-a.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode %v", got)
	}
}

// A write that cannot place the file where it belongs fails and leaves nothing
// beside it.
func TestAWriteThatCannotReplaceTheFilePreservesNothingPrior(t *testing.T) {
	dir := t.TempDir()
	occupied := filepath.Join(dir, "vault-a.json")
	if err := os.MkdirAll(filepath.Join(occupied, "inner"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := appstate.VaultFilesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Fatal("replaced a folder with a file")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "vault-a.json" {
		t.Errorf("the folder holds %v", entries)
	}
}

// A folder that cannot be written into fails the write.
func TestAFolderThatCannotBeSavedIntoFailsTheWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("a superuser writes into any folder")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Error(err)
		}
	})

	if err := appstate.VaultFilesAt(dir).Write(t.Context(), "vault-a", []byte("{}")); err == nil {
		t.Error("wrote into a read-only folder")
	}
}
