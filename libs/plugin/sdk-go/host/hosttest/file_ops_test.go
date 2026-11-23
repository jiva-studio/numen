package hosttest_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

const pdfBody = "%PDF-1.7\n\x00\xff\xfe binary \x80 tail"

func newDocuments() *hosttest.Vault {
	v := hosttest.New("Documents")
	v.WriteFile("Papers/Report.pdf", pdfBody)
	v.WriteFile("Papers/Notes/One.md", "one")
	v.WriteFile("Papers/Notes/Two.md", "two")
	return v
}

func TestMoveFileMovesAFileOfAnyKindWithItsBytes(t *testing.T) {
	v := newDocuments()

	got, err := v.MoveFile(t.Context(), "Papers/Report.pdf", "Archive/Old report.pdf", nil)

	if err != nil || got.Count != 1 {
		t.Fatalf("moved %+v, error = %v, want one file", got, err)
	}
	if content, there := v.ReadContent("Archive/Old report.pdf"); !there || content != pdfBody {
		t.Errorf("the target holds %q (there %v), want the bytes of the file", content, there)
	}
	if _, there := v.ReadContent("Papers/Report.pdf"); there {
		t.Error("the source is still there")
	}
}

func TestMoveFileMovesAFolderWithEveryFileInsideItAndCountsTheFiles(t *testing.T) {
	v := newDocuments()

	got, err := v.MoveFile(t.Context(), "Papers", "Archive/Papers", nil)

	if err != nil || got.Count != 3 {
		t.Fatalf("moved %+v, error = %v, want three files", got, err)
	}
	for _, name := range []string{"Archive/Papers/Report.pdf", "Archive/Papers/Notes/One.md", "Archive/Papers/Notes/Two.md"} {
		if _, there := v.ReadContent(name); !there {
			t.Errorf("%s is missing under the new folder", name)
		}
	}
	if _, there := v.ReadContent("Papers/Notes/One.md"); there {
		t.Error("a file is left under the old folder")
	}
}

func TestMoveFileRefusesAMissingSourceATakenTargetAndAPathOutsideTheVault(t *testing.T) {
	v := newDocuments()
	v.WriteFile("Archive/Taken.pdf", "x")

	for _, one := range []struct {
		name     string
		from, to string
		want     error
	}{
		{"missing source", "Papers/Gone.pdf", "Archive/Gone.pdf", host.ErrNotExist},
		{"taken target", "Papers/Report.pdf", "Archive/Taken.pdf", host.ErrAlreadyExists},
		{"source outside", "../Secret.pdf", "Archive/Secret.pdf", host.ErrOutside},
		{"target outside", "Papers/Report.pdf", "../Report.pdf", host.ErrOutside},
	} {
		if _, err := v.MoveFile(t.Context(), one.from, one.to, nil); !errors.Is(err, one.want) {
			t.Errorf("%s: error = %v, want %v", one.name, err, one.want)
		}
	}
	if content, there := v.ReadContent("Papers/Report.pdf"); !there || content != pdfBody {
		t.Error("a refused move changed the source")
	}
}

func TestMoveFileWithABaselineRefusesAFileThatChangedSinceAndMovesOneThatDidNot(t *testing.T) {
	v := newDocuments()
	seen := statOf(t, v, "Papers/Report.pdf")
	v.WriteFile("Papers/Report.pdf", pdfBody+" edited")

	if _, err := v.MoveFile(t.Context(), "Papers/Report.pdf", "Archive/Report.pdf", &seen); !errors.Is(err, host.ErrStale) {
		t.Errorf("a changed file: error = %v, want ErrStale", err)
	}
	if _, there := v.ReadContent("Papers/Report.pdf"); !there {
		t.Error("a refused move took the file")
	}
	current := statOf(t, v, "Papers/Report.pdf")
	if _, err := v.MoveFile(t.Context(), "Papers/Report.pdf", "Archive/Report.pdf", &current); err != nil {
		t.Errorf("an unchanged file: error = %v, want none", err)
	}
}

func TestRemoveFileMovesAFileToTheTrashAndDestroysNothing(t *testing.T) {
	v := newDocuments()

	got, err := v.RemoveFile(t.Context(), "Papers/Report.pdf", nil)

	if err != nil || got.Count != 1 {
		t.Fatalf("removed %+v, error = %v, want one file", got, err)
	}
	if _, there := v.ReadContent("Papers/Report.pdf"); there {
		t.Error("the file is still in the vault")
	}
	if content, there := v.ReadContent(".trash/Papers/Report.pdf"); !there || content != pdfBody {
		t.Errorf("the trash holds %q (there %v), want the bytes of the file", content, there)
	}
}

func TestRemoveFileTrashesAFolderWithEveryFileInsideItAndRefusesAMissingPath(t *testing.T) {
	v := newDocuments()

	got, err := v.RemoveFile(t.Context(), "Papers", nil)

	if err != nil || got.Count != 3 {
		t.Fatalf("removed %+v, error = %v, want three files", got, err)
	}
	if content, there := v.ReadContent(".trash/Papers/Notes/Two.md"); !there || content != "two" {
		t.Errorf("the trash holds %q (there %v) of a file under the folder", content, there)
	}
	if _, err := v.RemoveFile(t.Context(), "Papers", nil); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a second removal: error = %v, want ErrNotExist", err)
	}
	if _, err := v.RemoveFile(t.Context(), "../Secret.pdf", nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("a path outside: error = %v, want ErrOutside", err)
	}
}

func TestRemoveFileWithABaselineKeepsAFileThatChangedSince(t *testing.T) {
	v := newDocuments()
	seen := statOf(t, v, "Papers/Report.pdf")
	v.WriteFile("Papers/Report.pdf", pdfBody+" edited")

	_, err := v.RemoveFile(t.Context(), "Papers/Report.pdf", &seen)

	if !errors.Is(err, host.ErrStale) {
		t.Errorf("error = %v, want ErrStale", err)
	}
	if _, there := v.ReadContent("Papers/Report.pdf"); !there {
		t.Error("a refused removal took the file")
	}
}

func TestReadBytesReturnsTheBytesAsTheyAreWithTheStatAndHashOfTheFile(t *testing.T) {
	v := newDocuments()

	got, err := v.ReadBytes(t.Context(), "Papers/Report.pdf", 9, 4)

	if err != nil {
		t.Fatal(err)
	}
	if want := []byte(pdfBody)[9:13]; !bytes.Equal(got.Content, want) {
		t.Errorf("content = %x, want %x", got.Content, want)
	}
	if want := statOf(t, v, "Papers/Report.pdf"); got.Stat.Size != want.Size || got.Stat.Hash != want.Hash || got.Stat.Hash != domain.HashContent([]byte(pdfBody)) {
		t.Errorf("stat = %+v, want the stat of the whole file with its hash", got.Stat)
	}
}

func TestReadBytesPastTheEndReturnsWhatIsThereAndAtTheEndReturnsNone(t *testing.T) {
	v := newDocuments()
	size := int64(len(pdfBody))

	tail, err := v.ReadBytes(t.Context(), "Papers/Report.pdf", size-3, 100)
	if err != nil || !bytes.Equal(tail.Content, []byte(pdfBody)[size-3:]) {
		t.Errorf("a range past the end: %x, error = %v, want the last 3 bytes", tail.Content, err)
	}
	for _, start := range []int64{size, size + 50} {
		got, err := v.ReadBytes(t.Context(), "Papers/Report.pdf", start, 10)
		if err != nil || len(got.Content) != 0 || got.Stat.Size != size {
			t.Errorf("start %d: %d bytes, stat %+v, error = %v, want none and the stat", start, len(got.Content), got.Stat, err)
		}
	}
}

func TestReadBytesRefusesAFolderAMissingFileAnOutsidePathAndALengthOutsideTheLimits(t *testing.T) {
	v := newDocuments()

	for _, one := range []struct {
		name   string
		path   string
		start  int64
		length int64
		want   error
	}{
		{"a folder", "Papers", 0, 10, host.ErrNotExist},
		{"a missing file", "Papers/Gone.pdf", 0, 10, host.ErrNotExist},
		{"outside the vault", "../Secret.pdf", 0, 10, host.ErrOutside},
		{"length zero", "Papers/Report.pdf", 0, 0, host.ErrInvalidRange},
		{"a negative length", "Papers/Report.pdf", 0, -1, host.ErrInvalidRange},
		{"over the limit", "Papers/Report.pdf", 0, host.MaxReadBytes + 1, host.ErrInvalidRange},
		{"a negative start", "Papers/Report.pdf", -1, 10, host.ErrInvalidRange},
	} {
		if _, err := v.ReadBytes(t.Context(), one.path, one.start, one.length); !errors.Is(err, one.want) {
			t.Errorf("%s: error = %v, want %v", one.name, err, one.want)
		}
	}
	if _, err := v.ReadBytes(t.Context(), "Papers/Report.pdf", 0, host.MaxReadBytes); err != nil {
		t.Errorf("a length at the limit: error = %v, want none", err)
	}
}

func TestPlaceFileMovesAStagedFileToTheVaultAndEmptiesTheStoreName(t *testing.T) {
	v := hosttest.New("Documents")
	v.Store().WriteFile("0a1b.part", []byte(pdfBody))

	stat, err := v.PlaceFile(t.Context(), "0a1b.part", "Papers/Report.pdf", domain.HashContent([]byte(pdfBody)), nil)

	if err != nil {
		t.Fatal(err)
	}
	if content, there := v.ReadContent("Papers/Report.pdf"); !there || content != pdfBody {
		t.Errorf("the vault holds %q (there %v), want the staged bytes", content, there)
	}
	if stat.Hash != domain.HashContent([]byte(pdfBody)) || stat.Size != int64(len(pdfBody)) {
		t.Errorf("stat = %+v, want the hash and size of the placed file", stat)
	}
	if _, err := v.Store().Read(t.Context(), "0a1b.part"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the staged file: error = %v, want ErrNotExist after the placement", err)
	}
}

func TestPlaceFileRefusesTheWrongHashAndChangesNeitherTheVaultNorTheStagedFile(t *testing.T) {
	v := hosttest.New("Documents")
	v.WriteFile("Papers/Report.pdf", "old")
	seen := statOf(t, v, "Papers/Report.pdf")
	v.Store().WriteFile("0a1b.part", []byte(pdfBody))

	_, err := v.PlaceFile(t.Context(), "0a1b.part", "Papers/Report.pdf", domain.HashContent([]byte("another")), &seen)

	if !errors.Is(err, host.ErrHashMismatch) {
		t.Errorf("error = %v, want ErrHashMismatch", err)
	}
	if content, _ := v.ReadContent("Papers/Report.pdf"); content != "old" {
		t.Errorf("the vault holds %q, want the old file", content)
	}
	if _, err := v.Store().Read(t.Context(), "0a1b.part"); err != nil {
		t.Errorf("the staged file: error = %v, want it kept", err)
	}
}

func TestPlaceFileWithoutABaselineNeedsAFreePathAndWithOneNeedsAnUnchangedFile(t *testing.T) {
	v := hosttest.New("Documents")
	v.WriteFile("Papers/Report.pdf", "old")
	seen := statOf(t, v, "Papers/Report.pdf")
	hash := domain.HashContent([]byte(pdfBody))
	v.Store().WriteFile("a.part", []byte(pdfBody))

	if _, err := v.PlaceFile(t.Context(), "a.part", "Papers/Report.pdf", hash, nil); !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("a taken path without a baseline: error = %v, want ErrAlreadyExists", err)
	}
	v.WriteFile("Papers/Report.pdf", "old, then edited")
	if _, err := v.PlaceFile(t.Context(), "a.part", "Papers/Report.pdf", hash, &seen); !errors.Is(err, host.ErrStale) {
		t.Errorf("a changed file: error = %v, want ErrStale", err)
	}
	current := statOf(t, v, "Papers/Report.pdf")
	if _, err := v.PlaceFile(t.Context(), "a.part", "Papers/Report.pdf", hash, &current); err != nil {
		t.Fatalf("an unchanged file: error = %v, want none", err)
	}
	if content, _ := v.ReadContent("Papers/Report.pdf"); content != pdfBody {
		t.Errorf("the vault holds %q, want the placed bytes", content)
	}
}

func TestPlaceFileRefusesAMissingStagedFileAndAPathOutsideTheVault(t *testing.T) {
	v := hosttest.New("Documents")
	v.Store().WriteFile("a.part", []byte("x"))
	hash := domain.HashContent([]byte("x"))

	if _, err := v.PlaceFile(t.Context(), "missing.part", "Papers/A.pdf", hash, nil); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a missing staged file: error = %v, want ErrNotExist", err)
	}
	if _, err := v.PlaceFile(t.Context(), "a.part", "../A.pdf", hash, nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("a path outside: error = %v, want ErrOutside", err)
	}
	if _, err := v.PlaceFile(t.Context(), "../a.part", "Papers/A.pdf", hash, nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("a name outside the store folder: error = %v, want ErrOutside", err)
	}
}

func statOf(tb testing.TB, v *hosttest.Vault, name string) domain.FileStat {
	tb.Helper()
	stat, err := v.StatFile(tb.Context(), name)
	if err != nil {
		tb.Fatal(err)
	}
	return stat
}
