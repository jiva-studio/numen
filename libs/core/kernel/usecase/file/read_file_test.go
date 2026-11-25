package file_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func TestAWholeFileIsReadWithTheFingerprintItHadBeforeItsBytesWereRead(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	u := file.NewReadFile(vaultreader.Readers{}, vaultwriter.VaultWriters{})

	first, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(first.Content) != "body" || first.Stat.Path != "samples/A.md" || first.Stat.Size != 4 {
		t.Errorf("read %q with %+v", first.Content, first.Stat)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "samples", "A.md"), []byte("longer body"), 0o600); err != nil {
		t.Fatal(err)
	}
	updated, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil || updated.Stat.Size != int64(len("longer body")) {
		t.Errorf("a read after the file changed had %+v (%v), want the new size", updated.Stat, err)
	}
}

func TestAFileThatIsNotThereIsNotReadAsAWholeFile(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	u := file.NewReadFile(vaultreader.Readers{}, vaultwriter.VaultWriters{})

	_, err := u.Execute(t.Context(), v, "gone.md")

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAWholeFileIsReadWithTheHashOfItsBytesAndTheVersionOfTheVaultBeforeTheRead(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	u := file.NewReadFile(vaultreader.Readers{}, writers)
	before, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := writers.OpenNoteWriter(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "samples/A.md", []byte("next"), domain.FileStat{Hash: before.Stat.Hash}); err != nil {
		t.Fatal(err)
	}

	after, err := u.Execute(t.Context(), v, "samples/A.md")

	if err != nil {
		t.Fatal(err)
	}
	if before.Stat.Hash != domain.HashContent([]byte("body")) || after.Stat.Hash != domain.HashContent(after.Content) {
		t.Errorf("the hashes were %q and %q", before.Stat.Hash, after.Stat.Hash)
	}
	if after.Stat.Version != before.Stat.Version+1 {
		t.Errorf("the version was %d before one write and %d after it", before.Stat.Version, after.Stat.Version)
	}
}
