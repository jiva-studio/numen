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

	content, stat, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "body" || stat.Path != "samples/A.md" || stat.Size != 4 {
		t.Errorf("read %q with %+v", content, stat)
	}
	if err := os.WriteFile(filepath.Join(v.Path, "samples", "A.md"), []byte("longer body"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, now, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil || now.Size != int64(len("longer body")) {
		t.Errorf("a read after the file changed had %+v (%v), want the new size", now, err)
	}
}

func TestAFileThatIsNotThereIsNotReadAsAWholeFile(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	u := file.NewReadFile(vaultreader.Readers{}, vaultwriter.VaultWriters{})

	_, _, err := u.Execute(t.Context(), v, "gone.md")

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a file that is not there was read with %v", err)
	}
}

func TestAWholeFileIsReadWithTheHashOfItsBytesAndTheVersionOfTheVaultBeforeTheRead(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	u := file.NewReadFile(vaultreader.Readers{}, writers)
	_, before, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := writers.OpenNoteWriter(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "samples/A.md", []byte("next"), domain.FileStat{Hash: before.Hash}); err != nil {
		t.Fatal(err)
	}

	content, after, err := u.Execute(t.Context(), v, "samples/A.md")

	if err != nil {
		t.Fatal(err)
	}
	if before.Hash != domain.HashContent([]byte("body")) || after.Hash != domain.HashContent(content) {
		t.Errorf("the hashes were %q and %q", before.Hash, after.Hash)
	}
	if after.Version != before.Version+1 {
		t.Errorf("the version was %d before one write and %d after it", before.Version, after.Version)
	}
}
