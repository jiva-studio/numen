package syncvault_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/tools/syncserver/wiretest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/syncvault"
)

const (
	source domain.VaultID = "Test"
	target domain.VaultID = "Copy"
)

type pair struct {
	server *synctest.Server
	a, b   *synctest.Computer
}

func newPair(t *testing.T, cfg fakeserver.Config) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, cfg)
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	return &pair{server: server, a: a, b: b}
}

func sync(t *testing.T, c *synctest.Computer, id domain.VaultID) {
	t.Helper()
	if err := c.Sync(t.Context(), id); err != nil {
		t.Fatalf("pass over %s: %v", id, err)
	}
}

func wantNote(t *testing.T, c *synctest.Computer, id domain.VaultID, path, text string) {
	t.Helper()
	if got, ok := c.ReadNote(id, path); !ok || got != text {
		t.Errorf("note %s of %s = %q (there: %v), want %q", path, id, got, ok, text)
	}
}

func TestSyncVault_ABrandNewNoteAppearsOnTheSecondComputer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "milk")
	row, ok := p.a.GetFileRow(t, source, "Shopping list.md")
	if !ok || row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 {
		t.Errorf("row of the first computer = %+v (there: %v), want the hash of milk as base and the head revision", row, ok)
	}
}

func TestSyncVault_AChangedNoteChangesOnTheSecondComputer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	first := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))[0]
	p.a.WriteNote(source, "Shopping list.md", "bread")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "bread")
	head := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))
	if len(head) != 1 || !bytes.Equal(head[0].GetRevision().GetParentRev(), first.GetRevId()) {
		t.Errorf("head = %v, want one revision that has the first revision as its parent", head)
	}
}

func TestSyncVault_ANoteInAFolderKeepsItsFolder(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Food/Shopping list.md", "eggs")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Food/Shopping list.md", "eggs")
}

func TestSyncVault_ANoteFromTheSecondComputerAppearsOnTheFirst(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.b.WriteNote(target, "Plan.md", "tea")

	sync(t, p.b, target)
	sync(t, p.a, source)

	wantNote(t, p.a, source, "Plan.md", "tea")
	if p.a.HasItemKey(healthkey.Create("unverified-changes", string(source))) {
		t.Error("the first computer reports unverified changes for a device it learned from the server")
	}
}

func TestSyncVault_UploadsTheNotesThatWereThereBeforeTheVaultWasConnected(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.WriteNote(source, "Shopping list.md", "milk")
	a.WriteNote(source, "Plan.md", "tea")
	a.Connect(t, source)
	b := synctest.NewSecondComputer(t, server, clock, string(target), words)
	b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)

	sync(t, a, source)
	sync(t, b, target)

	wantNote(t, b, target, "Shopping list.md", "milk")
	wantNote(t, b, target, "Plan.md", "tea")
}

func TestSyncVault_DoesNotSendANoteBackThatItWroteItself(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.b.Vaults.With(target).Reindex()
	reserved := p.server.Fake.CountCalls(syncv1connect.BlobServiceReserveBlobsProcedure)
	committed := p.server.Fake.CountCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)

	sync(t, p.b, target)

	if got := p.server.Fake.CountCalls(syncv1connect.BlobServiceReserveBlobsProcedure); got != reserved {
		t.Errorf("reservations grew from %d to %d, want none for a note the pull wrote", reserved, got)
	}
	if got := p.server.Fake.CountCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure); got != committed {
		t.Errorf("commits grew from %d to %d, want none for a note the pull wrote", committed, got)
	}
}

func TestSyncVault_AFileThatIsNotANoteAppearsOnTheSecondComputerAsItIs(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Papers/Report.pdf", "%PDF-1.7 bytes")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Papers/Report.pdf", "%PDF-1.7 bytes")
	row, ok := p.b.GetFileRow(t, target, "Papers/Report.pdf")
	if !ok || row.BaseHash != note.Hash([]byte("%PDF-1.7 bytes")) {
		t.Errorf("row of the second computer = %+v (there: %v), want the hash of the file as base", row, ok)
	}
}

func TestSyncVault_SendsNoFileThatIsNotThePersons(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Papers/Report.pdf.part", "half")
	p.a.WriteNote(source, "Papers/~$Report.docx", "lock")
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "milk")
	for _, path := range []string{"Papers/Report.pdf.part", "Papers/~$Report.docx"} {
		if _, ok := p.b.ReadNote(target, path); ok {
			t.Errorf("%s reached the second computer", path)
		}
	}
	if got := len(p.a.ReadChanges(t, p.a.GetRemoteID(t, source))); got != 1 {
		t.Errorf("server holds %d revisions, want 1", got)
	}
}

func TestSyncVault_ShowsTheServerNoPlainTextOfTheVault(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)

	wire := bytes.Clone(p.server.GetStoredBlobs(t))
	for _, entry := range p.a.ReadCatalog(t) {
		raw, err := wiretest.EncodeMessage(entry)
		if err != nil {
			t.Fatal(err)
		}
		wire = append(wire, raw...)
	}
	for _, change := range p.a.ReadChanges(t, p.a.GetRemoteID(t, source)) {
		raw, err := wiretest.EncodeMessage(change)
		if err != nil {
			t.Fatal(err)
		}
		wire = append(wire, raw...)
	}

	for _, plain := range []string{"milk", "Shopping list", "Test", "Copy"} {
		if bytes.Contains(wire, []byte(plain)) {
			t.Errorf("the server holds %q as plain text", plain)
		}
	}
}

func TestSyncVault_CountsTheNotesChangedOnTwoComputers(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	for _, path := range []string{"One.md", "Two.md"} {
		p.a.WriteNote(source, path, "milk")
	}
	sync(t, p.a, source)
	sync(t, p.b, target)
	for _, path := range []string{"One.md", "Two.md"} {
		p.a.WriteNote(source, path, "bread")
		p.b.WriteNote(target, path, "eggs")
	}
	sync(t, p.a, source)

	sync(t, p.b, target)

	if !p.b.HasItem("2 notes changed on two computers") {
		t.Errorf("health items = %+v, want 2 notes changed on two computers", p.b.Health.List())
	}
}

func TestSyncVault_ANoteMadeWhileTheServerIsAwayArrivesLater(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.server.SetDown(true)
	p.a.WriteNote(source, "Shopping list.md", "milk")

	err := p.a.Sync(t.Context(), source)
	p.server.SetDown(false)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("pass with the server away = %v, want ErrUnreachable", err)
	}
	sync(t, p.a, source)
	sync(t, p.b, target)
	wantNote(t, p.b, target, "Shopping list.md", "milk")
}

func TestSyncVault_ConvergesAfterTheResponseOfACommitIsLost(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	p.server.LoseNextResponse("/CommitRevisions")

	err := p.a.Sync(t.Context(), source)
	sync(t, p.a, source)
	sync(t, p.b, target)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("pass with the response lost = %v, want ErrUnreachable", err)
	}
	head := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))
	if len(head) != 1 {
		t.Fatalf("server holds %d revisions, want 1", len(head))
	}
	row, _ := p.a.GetFileRow(t, source, "Shopping list.md")
	if row.BaseHash != note.Hash([]byte("milk")) || len(row.HeadRev) == 0 {
		t.Errorf("row = %+v, want the base and the head of the revision the server kept", row)
	}
	wantNote(t, p.b, target, "Shopping list.md", "milk")
}

func TestSyncVault_WarnsThatTheCloudStorageIsFullAndUploadsNothing(t *testing.T) {
	p := newPair(t, fakeserver.Config{QuotaBytes: 3000})
	p.a.WriteNote(source, "Shopping list.md", "milk")

	sync(t, p.a, source)

	if !p.a.HasItem("Cloud storage is full: changes stay on this device") {
		t.Errorf("health items = %+v, want the warning that the cloud storage is full", p.a.Health.List())
	}
	if got := len(p.a.ReadChanges(t, p.a.GetRemoteID(t, source))); got != 0 {
		t.Errorf("server holds %d revisions, want none", got)
	}
	wantNote(t, p.a, source, "Shopping list.md", "milk")
}

func TestSyncVault_FollowsTheNewNameOfTheVault(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	if _, err := p.a.Vaults.RenameVault(t.Context(), source, "Work"); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)

	entries := p.a.ReadCatalog(t)
	if len(entries) != 2 || entries[1].GetCreatesVault() || p.a.OpenCatalogBody(t, entries[1]).Name != "Work" {
		t.Fatalf("catalog = %v, want a second entry that renames the vault to Work", entries)
	}
	listed, err := catalog.NewListRemoteVaults(p.b.CatalogDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "Work" || !listed[0].IsConnected {
		t.Errorf("remote vaults of the second computer = %+v, want Work, still connected", listed)
	}
}

func TestSyncVault_RefusesAVaultThatIsNotConnected(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.AddVault(t, "Plain")

	err := p.a.Sync(t.Context(), "Plain")

	if !errors.Is(err, syncvault.ErrNotConnected) {
		t.Errorf("error = %v, want ErrNotConnected", err)
	}
}

func TestSyncVault_KeepsOneHeadForANoteChangedInManyPasses(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	for i := range 3 {
		p.a.WriteNote(source, "Shopping list.md", fmt.Sprintf("round %d", i))
		sync(t, p.a, source)
		sync(t, p.b, target)
		wantNote(t, p.b, target, "Shopping list.md", fmt.Sprintf("round %d", i))
	}

	if got := len(p.a.ReadChanges(t, p.a.GetRemoteID(t, source))); got != 1 {
		t.Errorf("server holds %d heads, want 1 for one note", got)
	}
}

func TestSyncVault_APassOverAnotherVaultKeepsTheWarningAboutFullStorage(t *testing.T) {
	p := newPair(t, fakeserver.Config{QuotaBytes: 3000})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	p.a.AddVault(t, "Plain")
	p.a.Connect(t, "Plain")
	sync(t, p.a, source)

	sync(t, p.a, "Plain")

	if !p.a.HasItem("Cloud storage is full: changes stay on this device") {
		t.Errorf("health items = %+v, want the warning about full storage to stay for the vault that is full", p.a.Health.List())
	}
}

func sealKeyForAnotherIdentity(t *testing.T, c *synctest.Computer, id domain.VaultID) {
	t.Helper()
	link := c.Link(t, id)
	link.SealedKey = bytes.Repeat([]byte{7}, len(link.SealedKey))
	if err := c.State.WriteLink(t.Context(), id, link); err != nil {
		t.Fatal(err)
	}
}

func TestSyncVault_AVaultWhoseKeyBelongsToAnotherIdentityIsRegisteredAgainAndUploads(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	old := p.a.Link(t, source).RemoteVaultID
	sealKeyForAnotherIdentity(t, p.a, source)
	p.a.WriteNote(source, "a.md", "kept")

	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatalf("pass over the vault: %v", err)
	}
	sync(t, p.a, source)

	if got := p.a.Link(t, source).RemoteVaultID; got == old {
		t.Error("the vault kept its old registration")
	}
	p.a.GetVaultKey(t, source)
	if len(p.a.ReadFileRows(t, source)) == 0 {
		t.Error("the note was not uploaded after the vault was registered again")
	}
	if p.a.HasItem("This vault was synced with another identity, so it is synced again with the current one.") {
		t.Error("the item stays after a pass that succeeded")
	}
}

func TestConnectVault_SyncThisVaultRegistersAgainWithAFreshKey(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	old := p.a.Link(t, source).RemoteVaultID
	sealKeyForAnotherIdentity(t, p.a, source)

	p.a.Connect(t, source)

	if got := p.a.Link(t, source).RemoteVaultID; got == old {
		t.Error("the vault kept its old registration")
	}
	p.a.GetVaultKey(t, source)
}
