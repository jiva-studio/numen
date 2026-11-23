package share_test

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func TestKeys_HoldOneKeyForEachEpochAndReplaceTheKeyOfAnEpoch(t *testing.T) {
	keys := share.Keys{}.Add(0, newKey(1)).Add(1, newKey(2)).Add(0, newKey(3))

	if got, ok := keys.Get(0); !ok || got != newKey(3) {
		t.Errorf("key of epoch 0 = %x (%v), want the replacing key", got, ok)
	}
	if got, ok := keys.Get(1); !ok || got != newKey(2) {
		t.Errorf("key of epoch 1 = %x (%v), want the second key", got, ok)
	}
	if _, ok := keys.Get(2); ok {
		t.Error("a key of an epoch nobody added")
	}
	if len(keys) != 2 {
		t.Errorf("%d keys, want 2", len(keys))
	}
}

func TestOwnerBody_ReadsBackWhatItWroteAndKeepsRemovedReaders(t *testing.T) {
	removed := time.Date(2026, time.November, 24, 9, 0, 0, 0, time.UTC)
	body := share.OwnerBody{
		Folder: "Life/Food", Keys: share.Keys{}.Add(0, newKey(1)),
		Readers: []share.Reader{
			{Root: bytes.Repeat([]byte{1}, 32), Chain: []byte{9}, Label: "Anna", Fingerprint: "0F3A", AddedAt: removed.Add(-time.Hour)},
			{Root: bytes.Repeat([]byte{2}, 32), Chain: []byte{8}, Label: "Bo", Fingerprint: "A1B2", AddedAt: removed.Add(-time.Hour), RemovedAt: removed},
		},
	}

	raw, err := body.Encode()
	if err != nil {
		t.Fatal(err)
	}
	read, err := share.ParseOwnerBody(raw)

	if err != nil {
		t.Fatal(err)
	}
	if read.Folder != "Life/Food" || len(read.Readers) != 2 || !read.Readers[0].IsActive() || read.Readers[1].IsActive() ||
		!read.Readers[1].RemovedAt.Equal(removed) || read.Readers[1].GetState() != share.StateRemoved {
		t.Errorf("body = %+v, want the folder and both readers, the second removed", read)
	}
	if key, ok := read.Keys.Get(0); !ok || key != newKey(1) {
		t.Errorf("key of epoch 0 = %x (%v), want the key that was sealed", key, ok)
	}
}

func TestCheckReader_RefusesAnActiveReaderAndALabelOfAnotherPersonUnlessConfirmed(t *testing.T) {
	anna, bo := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	readers := []share.Reader{
		{Root: anna, Label: "Anna"},
		{Root: bo, Label: "Bo", RemovedAt: time.Date(2026, time.November, 24, 9, 0, 0, 0, time.UTC)},
	}
	for name, c := range map[string]struct {
		root        []byte
		label       string
		isConfirmed bool
		want        error
	}{
		"an active reader":                  {root: anna, label: "Anna", want: share.ErrAlreadyReader},
		"an active reader with a new label": {root: anna, label: "Annie", want: share.ErrAlreadyReader},
		"a label of another person":         {root: bytes.Repeat([]byte{3}, 32), label: "anna", want: share.ErrChangedIdentity},
		"the label confirmed":               {root: bytes.Repeat([]byte{3}, 32), label: "Anna", isConfirmed: true},
		"a removed reader again":            {root: bo, label: "Bo"},
		"a new person":                      {root: bytes.Repeat([]byte{3}, 32), label: "Cy"},
	} {
		if err := share.CheckReader(readers, c.root, c.label, c.isConfirmed); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestFindRoot_SkipsAClosedShareAndFindsTheShareOfAFileByItsFolder(t *testing.T) {
	roots := []share.Root{
		{ID: []byte{1}, Folder: "Food", IsClosed: true},
		{ID: []byte{2}, Folder: "Life/Plans"},
		{ID: []byte{3}, Folder: "Food"},
	}

	byFolder, isFolder := share.FindRoot(roots, "food/")
	byFile, isFile := share.FindRootOf(roots, "Life/Plans/Trip.md")
	closedFile, isClosed := share.FindRootOf(roots[:1], "Food/Soup.md")
	byID, isID := share.FindRootByID(roots, []byte{1})

	if !isFolder || byFolder.ID[0] != 3 {
		t.Errorf("share of the folder = %v (%v), want the open share 3", byFolder.ID, isFolder)
	}
	if !isFile || byFile.ID[0] != 2 {
		t.Errorf("share of the file = %v (%v), want share 2", byFile.ID, isFile)
	}
	if isClosed {
		t.Errorf("a file of a closed share belongs to share %v", closedFile.ID)
	}
	if !isID || !byID.IsClosed {
		t.Errorf("share by id = %+v (%v), want the closed share", byID, isID)
	}
	if got := share.ListFolders(roots); len(got) != 2 {
		t.Errorf("folders = %v, want the two open ones", got)
	}
}

func TestCleanName_RemovesSeparatorFromVaultName(t *testing.T) {
	for _, one := range []struct{ name, want string }{{"Food", "Food"}, {"a/b", "a_b"}, {"", "Shared"}, {"..", "Shared"}, {" Food ", "Food"}} {
		if got := share.CleanName(one.name); got != one.want {
			t.Errorf("CleanName(%q) = %q, want %q", one.name, got, one.want)
		}
	}
}
