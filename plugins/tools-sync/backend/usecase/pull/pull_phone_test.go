package pull_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/layer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
)

const (
	guidePath = "Books/Guide.pdf"
	heavySize = 5 << 20
)

func newPhonePair(t *testing.T) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, source, words)
	a.Connect(t, source)
	b := synctest.NewPhone(t, server, clock, target, words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &pair{server: server, a: a, b: b, vault: newVault(t, a)}
}

func getHeavy() string { return strings.Repeat("g", heavySize) }

func countBlobURLCalls(p *pair) int {
	return p.server.CountCalls(syncv1connect.BlobServiceGetBlobUrlsProcedure)
}

func TestPullChanges_ListsAHeavyFileAsAbsentAndDownloadsNothingOnAPhone(t *testing.T) {
	p := newPhonePair(t)
	p.push(t, guidePath, getHeavy(), nil)
	before := countBlobURLCalls(p)

	result := p.pull(t, trustAll)

	listed := p.b.ListAbsent(t, target)
	if len(listed) != 1 || listed[0].Path != guidePath || listed[0].Size != heavySize || listed[0].Hash != note.Hash([]byte(getHeavy())) {
		t.Errorf("absent files = %+v, want %s with its size and hash", listed, guidePath)
	}
	if after := countBlobURLCalls(p); after != before {
		t.Errorf("blob URL requests grew from %d to %d, want no download of a file on tap", before, after)
	}
	if _, _, err := p.b.Vaults.With(target).Read(t.Context(), guidePath); !errors.Is(err, host.ErrFileAbsent) {
		t.Errorf("read = %v, want ErrFileAbsent", err)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none", result.Conflicts)
	}
}

func TestPullChanges_RecordsTheBaseOfAnAbsentFile(t *testing.T) {
	p := newPhonePair(t)
	p.push(t, guidePath, getHeavy(), nil)

	p.pull(t, trustAll)

	row, ok := p.b.GetFileRow(t, target, guidePath)
	if !ok || row.BaseHash != note.Hash([]byte(getHeavy())) || len(row.HeadRev) == 0 || len(row.FileID) == 0 {
		t.Errorf("row = %+v (there: %v), want the hash as base, the head and the file id", row, ok)
	}
}

func TestPullChanges_DownloadsANoteOfAnySizeAtOnceOnAPhone(t *testing.T) {
	p := newPhonePair(t)
	long := strings.Repeat("milk and bread. ", 1<<18)
	p.push(t, "Journal.md", long, nil)

	p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, "Journal.md"); !ok || text != long {
		t.Errorf("note of %d bytes (there: %v), want the whole text", len(text), ok)
	}
	if p.b.IsAbsent(t, target, "Journal.md") {
		t.Error("a note is listed as absent")
	}
}

func TestPullChanges_DownloadsAnImageUpToTheLimitAtOnceOnAPhone(t *testing.T) {
	p := newPhonePair(t)
	small := strings.Repeat("p", layer.AtOnceImageLimit)
	p.push(t, "Pictures/Lake.png", small, nil)

	p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, "Pictures/Lake.png"); !ok || text != small {
		t.Errorf("image of %d bytes (there: %v), want the whole file", len(text), ok)
	}
}

func TestPullChanges_ListsAnImageOverTheLimitAsAbsentOnAPhone(t *testing.T) {
	p := newPhonePair(t)
	p.push(t, "Pictures/Big.png", strings.Repeat("p", layer.AtOnceImageLimit+1), nil)

	p.pull(t, trustAll)

	if !p.b.IsAbsent(t, target, "Pictures/Big.png") {
		t.Errorf("absent files = %+v, want the large image", p.b.ListAbsent(t, target))
	}
}

func TestPullChanges_DownloadsAHeavyFileAtOnceWhenOnDemandIsOff(t *testing.T) {
	p := newPair(t)
	p.push(t, guidePath, getHeavy(), nil)

	p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, guidePath); !ok || text != getHeavy() {
		t.Errorf("file of %d bytes (there: %v), want the whole file", len(text), ok)
	}
	if len(p.b.ListAbsent(t, target)) != 0 {
		t.Error("a device with on demand off listed a file as absent")
	}
}

func TestPullChanges_MovesTheEntryOfAnAbsentFile(t *testing.T) {
	p := newPhonePair(t)
	first := p.push(t, guidePath, getHeavy(), nil)
	p.pull(t, trustAll)
	p.push(t, "Archive/Guide.pdf", getHeavy(), &first)

	p.pull(t, trustAll)

	if !p.b.IsAbsent(t, target, "Archive/Guide.pdf") || p.b.IsAbsent(t, target, guidePath) {
		t.Errorf("absent files = %+v, want the file in Archive only", p.b.ListAbsent(t, target))
	}
	if row, _ := p.b.GetFileRow(t, target, "Archive/Guide.pdf"); row.BaseHash != note.Hash([]byte(getHeavy())) {
		t.Errorf("base = %s, want the hash kept", row.BaseHash)
	}
}

func TestPullChanges_ListsTheNewHeadOfAnAbsentFileAndDownloadsNothing(t *testing.T) {
	p := newPhonePair(t)
	first := p.push(t, guidePath, getHeavy(), nil)
	p.pull(t, trustAll)
	newer := strings.Repeat("h", heavySize)
	p.push(t, guidePath, newer, &first)
	before := countBlobURLCalls(p)

	result := p.pull(t, trustAll)

	listed := p.b.ListAbsent(t, target)
	if len(listed) != 1 || listed[0].Hash != note.Hash([]byte(newer)) {
		t.Errorf("absent files = %+v, want the entry of the new head", listed)
	}
	if after := countBlobURLCalls(p); after != before {
		t.Errorf("blob URL requests grew from %d to %d, want no download of a change nobody opened", before, after)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none: an absent file equals its base", result.Conflicts)
	}
}

func TestPullChanges_EvictsAndListsADownloadedFileThatGrewPastTheLimit(t *testing.T) {
	p := newPhonePair(t)
	first := p.push(t, "Pictures/Lake.png", strings.Repeat("p", 1<<10), nil)
	p.pull(t, trustAll)
	p.push(t, "Pictures/Lake.png", strings.Repeat("q", layer.AtOnceImageLimit+1), &first)

	p.pull(t, trustAll)

	if !p.b.IsAbsent(t, target, "Pictures/Lake.png") {
		t.Errorf("absent files = %+v, want the grown image", p.b.ListAbsent(t, target))
	}
	if _, ok := p.b.ReadNote(target, "Pictures/Lake.png"); ok {
		t.Error("the old bytes are still on the phone")
	}
}

func newVault(t *testing.T, a *synctest.Computer) transfer.Vault {
	t.Helper()
	return transfer.Vault{ID: a.GetRemoteID(t, source), Key: a.GetVaultKey(t, source)}
}

func TestPullChanges_KeepsTheFilesOnDiskWhenOnDemandTurnsOnAndListsOnlyNewHeavyFiles(t *testing.T) {
	p := newPair(t)
	p.push(t, guidePath, getHeavy(), nil)
	p.pull(t, trustAll)
	p.b.LazyHeavy = true
	p.push(t, "Books/Atlas.pdf", getHeavy(), nil)

	p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, guidePath); !ok || text != getHeavy() {
		t.Errorf("file already on disk = %d bytes (there: %v), want it kept", len(text), ok)
	}
	if listed := p.b.ListAbsent(t, target); len(listed) != 1 || listed[0].Path != "Books/Atlas.pdf" {
		t.Errorf("absent files = %+v, want only the new heavy file", listed)
	}
}

func TestPullChanges_ListsNothingNewAfterOnDemandTurnsOffAndTheAbsentFileStaysFetchable(t *testing.T) {
	p := newPhonePair(t)
	p.push(t, guidePath, getHeavy(), nil)
	p.pull(t, trustAll)
	p.b.LazyHeavy = false
	p.push(t, "Books/Atlas.pdf", getHeavy(), nil)

	p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, "Books/Atlas.pdf"); !ok || text != getHeavy() {
		t.Errorf("new file = %d bytes (there: %v), want it downloaded at once", len(text), ok)
	}
	if err := fetchfile.NewFetchFile(p.b.FetchDeps()).Execute(t.Context(), target, guidePath); err != nil {
		t.Fatalf("fetch of the absent file = %v, want nil", err)
	}
	if text, ok := p.b.ReadNote(target, guidePath); !ok || text != getHeavy() || p.b.IsAbsent(t, target, guidePath) {
		t.Errorf("fetched file = %d bytes (there: %v), want the whole file placed", len(text), ok)
	}
}
