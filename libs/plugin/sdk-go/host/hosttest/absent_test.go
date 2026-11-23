package hosttest_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

const (
	guidePath    = "Books/Guide.pdf"
	guideContent = "the guide, page one"
)

func getHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func getGuide() host.AbsentFile {
	return host.AbsentFile{
		Path: guidePath, Size: int64(len(guideContent)), ModTime: time.Date(2026, time.November, 20, 9, 30, 0, 0, time.UTC),
		Hash: getHash(guideContent),
	}
}

func putGuide(t *testing.T, v *hosttest.Vault) {
	t.Helper()
	if err := v.PutAbsentFiles(t.Context(), []host.AbsentFile{getGuide()}); err != nil {
		t.Fatal(err)
	}
}

func TestListAbsentFilesReturnsTheFilesPut(t *testing.T) {
	v := hosttest.New("garden")
	putGuide(t, v)
	got, err := v.ListAbsentFiles(t.Context())
	if err != nil || len(got) != 1 || got[0].Path != guidePath || got[0].Hash != getHash(guideContent) {
		t.Fatalf("got %+v, %v, want the entry of %s", got, err, guidePath)
	}
}

func TestReadOfAnAbsentFileReturnsErrFileAbsent(t *testing.T) {
	v := hosttest.New("garden")
	putGuide(t, v)
	if _, _, err := v.Read(t.Context(), guidePath); !errors.Is(err, host.ErrFileAbsent) {
		t.Fatalf("got %v, want ErrFileAbsent", err)
	}
}

func TestPutAbsentFilesRefusesAPathWithAFileOfAnotherHash(t *testing.T) {
	v := hosttest.New("garden")
	v.File(guidePath, "my own notes in a file")
	if err := v.PutAbsentFiles(t.Context(), []host.AbsentFile{getGuide()}); !errors.Is(err, host.ErrFilePresent) {
		t.Fatalf("got %v, want ErrFilePresent", err)
	}
}

func TestPutAbsentFilesReportsNoChangeInTheFeed(t *testing.T) {
	v := hosttest.New("garden")
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	putGuide(t, v)
	if page := readFeed(t, v, start); len(page.Changes) != 0 {
		t.Fatalf("got %v, want no change for a listed file", describe(page.Changes))
	}
}

func TestFillFilePlacesTheStoreFileAndReportsNoChange(t *testing.T) {
	v := hosttest.New("garden")
	putGuide(t, v)
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	v.Store().File("guide.part", []byte(guideContent))
	if err := v.FillFile(t.Context(), guidePath, "guide.part"); err != nil {
		t.Fatal(err)
	}
	if got, isThere := v.Content(guidePath); !isThere || got != guideContent {
		t.Fatalf("got %q, %v, want the bytes of the store file", got, isThere)
	}
	if listed, err := v.ListAbsentFiles(t.Context()); err != nil || len(listed) != 0 {
		t.Fatalf("got %+v, %v, want the entry gone", listed, err)
	}
	if page := readFeed(t, v, start); len(page.Changes) != 0 {
		t.Fatalf("got %v, want a fill that the feed does not report", describe(page.Changes))
	}
}

func TestFillFileWithOtherBytesReturnsErrHashMismatchAndPlacesNothing(t *testing.T) {
	v := hosttest.New("garden")
	putGuide(t, v)
	v.Store().File("guide.part", []byte("damaged"))
	if err := v.FillFile(t.Context(), guidePath, "guide.part"); !errors.Is(err, host.ErrHashMismatch) {
		t.Fatalf("got %v, want ErrHashMismatch", err)
	}
	if _, isThere := v.Content(guidePath); isThere {
		t.Fatal("a file is at the path after a mismatch")
	}
}

func TestEvictFileDeletesTheBytesListsTheFileAndReportsNoRemoval(t *testing.T) {
	v := hosttest.New("garden")
	v.File(guidePath, guideContent)
	start := readFeed(t, v, host.FileChangeCursor{}).Next
	if err := v.EvictFile(t.Context(), guidePath, getHash(guideContent)); err != nil {
		t.Fatal(err)
	}
	if _, isThere := v.Content(guidePath); isThere {
		t.Fatal("the bytes are still held")
	}
	if listed, err := v.ListAbsentFiles(t.Context()); err != nil || len(listed) != 1 || listed[0].Path != guidePath {
		t.Fatalf("got %+v, %v, want the file listed", listed, err)
	}
	if page := readFeed(t, v, start); len(page.Changes) != 0 {
		t.Fatalf("got %v, want an eviction that is no removal", describe(page.Changes))
	}
}

func TestEvictFileOfAFileWithChangesReturnsErrChangedAndKeepsIt(t *testing.T) {
	v := hosttest.New("garden")
	v.File(guidePath, guideContent+", then my notes")
	if err := v.EvictFile(t.Context(), guidePath, getHash(guideContent)); !errors.Is(err, host.ErrChanged) {
		t.Fatalf("got %v, want ErrChanged", err)
	}
	if _, isThere := v.Content(guidePath); !isThere {
		t.Fatal("a file with changes lost its bytes")
	}
}
