package pull_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

const (
	source = "Test"
	target = "Copy"
)

type pair struct {
	server *synctest.Server
	a, b   *synctest.Computer
	vault  transfer.Vault
}

func newPair(t *testing.T) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, source, words)
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, target, words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &pair{server: server, a: a, b: b, vault: transfer.Vault{ID: a.GetRemoteID(t, source), Key: a.GetVaultKey(t, source)}}
}

func (p *pair) push(t *testing.T, path, text string, previous *transfer.Outcome) transfer.Outcome {
	t.Helper()
	file := transfer.File{Path: path, Content: []byte(text)}
	if previous != nil {
		file.FileID, file.ParentRev = previous.FileID, previous.RevID
	}
	outcomes, err := commit.NewCommitRevisions(p.a.CommitDeps()).Execute(t.Context(), transfer.CommitInput{Vault: p.vault, Files: []transfer.File{file}})
	if err != nil {
		t.Fatal(err)
	}
	if outcomes[0].Status != transfer.Committed {
		t.Fatalf("push of %s = %+v, want committed", path, outcomes[0])
	}
	return outcomes[0]
}

func trustAll([]byte) bool { return true }

func (p *pair) pull(t *testing.T, authors func([]byte) bool) transfer.PullResult {
	t.Helper()
	result, err := pull.NewPullChanges(p.b.PullDeps()).Execute(t.Context(), transfer.PullInput{VaultID: target, IsAuthor: authors})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestPullChanges_WritesANewNoteInItsFolder(t *testing.T) {
	p := newPair(t)
	p.push(t, "Food/Shopping list.md", "eggs", nil)

	result := p.pull(t, trustAll)

	if text, ok := p.b.ReadNote(target, "Food/Shopping list.md"); !ok || text != "eggs" {
		t.Errorf("note = %q (there: %v), want eggs", text, ok)
	}
	if !slices.Equal(result.Applied, []string{"Food/Shopping list.md"}) || len(result.Conflicts) != 0 || result.Unverified != 0 {
		t.Errorf("result = %+v, want the one note applied", result)
	}
}

func TestPullChanges_RecordsTheBaseTheHeadAndThePositionOnTheServer(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)

	p.pull(t, trustAll)

	row, ok := p.b.GetFileRow(t, target, "Shopping list.md")
	if !ok || row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 || len(row.FileID) == 0 {
		t.Errorf("row = %+v (there: %v), want the hash of milk as base, the head and the file id", row, ok)
	}
	if link := p.b.Link(t, target); link.ServerSeq == 0 {
		t.Errorf("link = %+v, want the position on the server moved on", link)
	}
}

func TestPullChanges_ReplacesANoteThatEqualsItsBase(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Shopping list.md", "milk", nil)
	p.pull(t, trustAll)
	p.push(t, "Shopping list.md", "bread", &first)

	result := p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Shopping list.md"); text != "bread" {
		t.Errorf("note = %q, want bread", text)
	}
	if row, _ := p.b.GetFileRow(t, target, "Shopping list.md"); row.BaseHash != note.Hash([]byte("bread")) {
		t.Errorf("base = %s, want the hash of bread", row.BaseHash)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none", result.Conflicts)
	}
}

func TestPullChanges_AdoptsAHeadThatEqualsTheLocalNote(t *testing.T) {
	p := newPair(t)
	p.b.WriteNote(target, "Shopping list.md", "milk")
	p.push(t, "Shopping list.md", "milk", nil)

	result := p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Shopping list.md"); text != "milk" {
		t.Errorf("note = %q, want milk", text)
	}
	if row, ok := p.b.GetFileRow(t, target, "Shopping list.md"); !ok || row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 {
		t.Errorf("row = %+v (there: %v), want the hash as base and the head recorded", row, ok)
	}
	if len(result.Conflicts) != 0 {
		t.Errorf("conflicts = %v, want none for equal notes", result.Conflicts)
	}
}

func TestPullChanges_KeepsTheNoteWhenTheHostRefusesTheWrite(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Shopping list.md", "milk", nil)
	p.pull(t, trustAll)
	p.push(t, "Shopping list.md", "bread", &first)
	p.b.Vaults.With(target).RefuseWriting("Shopping list.md", host.ErrStale)

	result := p.pull(t, trustAll)

	if text, _ := p.b.ReadNote(target, "Shopping list.md"); text != "milk" {
		t.Errorf("note = %q, want milk, unchanged", text)
	}
	if !slices.Equal(result.Conflicts, []string{"Shopping list.md"}) {
		t.Errorf("conflicts = %v, want the note", result.Conflicts)
	}
	if row, _ := p.b.GetFileRow(t, target, "Shopping list.md"); row.BaseHash != note.Hash([]byte("milk")) {
		t.Errorf("base = %s, want the old base of milk back", row.BaseHash)
	}
}

func TestPullChanges_SkipsARevisionOfAnAuthorItDoesNotTrust(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)

	result := p.pull(t, func([]byte) bool { return false })

	if _, ok := p.b.ReadNote(target, "Shopping list.md"); ok {
		t.Error("a note of an author that is not trusted was written")
	}
	if result.Unverified != 1 || len(result.Applied) != 0 {
		t.Errorf("result = %+v, want one unverified revision and none applied", result)
	}
}

func TestPullChanges_RejectsABlobThatDiffersFromItsHash(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)
	p.server.SetBlobCorruption(true)

	_, err := pull.NewPullChanges(p.b.PullDeps()).Execute(t.Context(), transfer.PullInput{VaultID: target, IsAuthor: trustAll})

	if !errors.Is(err, pull.ErrCorruptBlob) {
		t.Errorf("error = %v, want ErrCorruptBlob", err)
	}
	if _, ok := p.b.ReadNote(target, "Shopping list.md"); ok {
		t.Error("a note was written from a damaged blob")
	}
	if link := p.b.Link(t, target); link.ServerSeq != 0 {
		t.Errorf("position on the server = %d, want it unchanged at 0", link.ServerSeq)
	}
}

func TestPullChanges_ReadsEveryPageOfChanges(t *testing.T) {
	p := newPair(t)
	files := make([]transfer.File, 205)
	for i := range files {
		files[i] = transfer.File{Path: fmt.Sprintf("n%03d.md", i), Content: []byte(fmt.Sprintf("note %d", i))}
	}
	if _, err := commit.NewCommitRevisions(p.a.CommitDeps()).Execute(t.Context(), transfer.CommitInput{Vault: p.vault, Files: files}); err != nil {
		t.Fatal(err)
	}

	result := p.pull(t, trustAll)

	if len(result.Applied) != 205 {
		t.Errorf("applied = %d notes, want 205", len(result.Applied))
	}
	if text, ok := p.b.ReadNote(target, "n204.md"); !ok || text != "note 204" {
		t.Errorf("last note = %q (there: %v), want note 204", text, ok)
	}
}

func TestPullChanges_DownloadsNothingWhenNothingChanged(t *testing.T) {
	p := newPair(t)
	p.push(t, "Shopping list.md", "milk", nil)
	p.pull(t, trustAll)
	before := p.server.Fake.CountCalls(syncv1connect.BlobServiceGetBlobUrlsProcedure)

	result := p.pull(t, trustAll)

	if after := p.server.Fake.CountCalls(syncv1connect.BlobServiceGetBlobUrlsProcedure); after != before {
		t.Errorf("blob URL requests grew from %d to %d, want no download", before, after)
	}
	if len(result.Applied) != 0 {
		t.Errorf("applied = %v, want none", result.Applied)
	}
}
