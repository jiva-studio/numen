package vaultabsent_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/vaultabsent"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultconfig"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const guidePath = "Books/Guide.pdf"

func getGuide() vault.AbsentFile {
	return vault.AbsentFile{
		Path: guidePath, Size: 5 << 20, Mtime: time.Date(2026, time.November, 20, 9, 30, 0, 0, time.UTC),
		Hash: "ab12", Provider: "tools-sync",
	}
}

func openPhone(t *testing.T, notes map[string]string, absent ...vault.AbsentFile) vaultport.VaultReader {
	t.Helper()
	v := domain.Vault{ID: "01VAULT0000000000000000000", Path: vaulttest.CreateVault(t, notes, nil)}
	db, err := index.Open(t.Context(), indexfile.GetPath(t), index.Unsynchronized())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	if err := db.AbsentFiles().PutAbsentFiles(t.Context(), v, absent); err != nil {
		t.Fatal(err)
	}
	reader, err := vaultabsent.NewReaders(vaultreader.Readers{Options: vaultconfig.Options{}}, db.AbsentFiles()).Open(v)
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func walk(t *testing.T, reader vaultport.VaultReader) map[string]domain.FileStat {
	t.Helper()
	got := map[string]domain.FileStat{}
	err := reader.WalkFiles(t.Context(), func(stat domain.FileStat) error {
		if _, isTwice := got[stat.Path]; isTwice {
			t.Errorf("the walk listed %s twice", stat.Path)
		}
		got[stat.Path] = stat
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestWalkFilesListsAnAbsentFileNextToTheFilesOfTheVault(t *testing.T) {
	reader := openPhone(t, map[string]string{"Shopping list.md": "milk"}, getGuide())
	got := walk(t, reader)
	if _, isNote := got["Shopping list.md"]; !isNote || len(got) != 2 {
		t.Fatalf("got %d files, want the note and the absent file", len(got))
	}
	guide := got[guidePath]
	if !guide.IsAbsent || guide.Size != 5<<20 || guide.Kind != domain.KindFile {
		t.Fatalf("got %+v, want an absent file of 5 MiB", guide)
	}
	if got["Shopping list.md"].IsAbsent {
		t.Fatal("the note is held and is listed as absent")
	}
}

func TestWalkFilesListsADownloadedFileOnce(t *testing.T) {
	reader := openPhone(t, map[string]string{guidePath: "bytes"}, getGuide())
	got := walk(t, reader)
	if len(got) != 1 || got[guidePath].IsAbsent {
		t.Fatalf("got %+v, want the held file once and not absent", got)
	}
}

func TestListShowsAnAbsentFileInItsFolder(t *testing.T) {
	reader := openPhone(t, map[string]string{"Books/Atlas.md": "maps"}, getGuide())
	entries, err := reader.List(t.Context(), "Books")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name)
		if entry.Path == guidePath && (!entry.IsAbsent || entry.Provider != "tools-sync") {
			t.Fatalf("got %+v, want an absent entry that names its provider", entry)
		}
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"Atlas.md", "Guide.pdf"}) {
		t.Fatalf("got %v, want both files", names)
	}
}

func TestListShowsAFolderOfOnlyAbsentFiles(t *testing.T) {
	reader := openPhone(t, map[string]string{"Shopping list.md": "milk"}, getGuide())
	entries, err := reader.List(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(entries, func(e domain.Entry) bool { return e.IsFolder && e.Name == "Books" }) {
		t.Fatalf("got %+v, want the folder Books", entries)
	}
}

func TestStatFileOfAnAbsentFileReturnsTheSizeAndTimeOfTheEntry(t *testing.T) {
	stat, err := openPhone(t, nil, getGuide()).StatFile(t.Context(), guidePath)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size != 5<<20 || !stat.ModTime.Equal(getGuide().Mtime) || !stat.IsAbsent {
		t.Fatalf("got %+v, want the size and time of the entry", stat)
	}
}

func TestReadsOfAnAbsentFileReturnErrFileAbsent(t *testing.T) {
	reader := openPhone(t, nil, getGuide())
	if _, err := reader.Read(t.Context(), guidePath); !errors.Is(err, vault.ErrFileAbsent) {
		t.Fatalf("Read returned %v, want ErrFileAbsent", err)
	}
	file, err := reader.Open(t.Context(), guidePath)
	if err == nil {
		t.Error(file.Close())
	}
	if !errors.Is(err, vault.ErrFileAbsent) {
		t.Fatalf("Open returned %v, want ErrFileAbsent", err)
	}
}

func TestReadOfAHeldFileIsNotChanged(t *testing.T) {
	got, err := openPhone(t, map[string]string{"Shopping list.md": "milk"}, getGuide()).Read(t.Context(), "Shopping list.md")
	if err != nil || string(got) != "milk" {
		t.Fatalf("got %q, %v, want milk", got, err)
	}
}

func TestStatFileOfAPathWithNothingReturnsNotExist(t *testing.T) {
	if _, err := openPhone(t, nil, getGuide()).StatFile(t.Context(), "Books/None.pdf"); err == nil || errors.Is(err, vault.ErrFileAbsent) {
		t.Fatalf("got %v, want a not-exist error", err)
	}
}
