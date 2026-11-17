package file_test

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func TestAStatCarriesTheHashOfTheContentAndTheVersionOfTheVault(t *testing.T) {
	v := vaulttest.NewVault(t, map[string]string{"samples/A.md": "body"})
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	u := file.NewHashFile(vaultreader.Readers{}, writers)
	before, err := u.Execute(t.Context(), v, "samples/A.md")
	if err != nil {
		t.Fatal(err)
	}
	writer, err := writers.OpenNoteWriter(v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(t.Context(), "samples/A.md", []byte("longer body"), domain.FileStat{Hash: before.Hash}); err != nil {
		t.Fatal(err)
	}

	after, err := u.Execute(t.Context(), v, "samples/A.md")

	if err != nil {
		t.Fatal(err)
	}
	if before.Hash != domain.HashContent([]byte("body")) || before.Size != 4 || before.Path != "samples/A.md" {
		t.Errorf("the stat was %+v", before)
	}
	if after.Hash != domain.HashContent([]byte("longer body")) || after.Version != before.Version+1 {
		t.Errorf("the stat after a write was %+v after %+v", after, before)
	}
}

func TestAStatOfAFileThatIsNotThereReturnsNotExist(t *testing.T) {
	v := vaulttest.NewVault(t, nil)

	_, err := file.NewHashFile(vaultreader.Readers{}, vaultwriter.VaultWriters{}).Execute(t.Context(), v, "gone.md")

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a stat of a file that is not there returned %v", err)
	}
}
