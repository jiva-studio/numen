package syncvault_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/catalog"
)

const (
	textWentBack       = "The sync server went back to an earlier state"
	textEpochUntrusted = "The restore announcement of the sync server is not signed by a known root key and was ignored"
	keyRunning         = "restore.running"
	keyWentBack        = "restore.wentBack"
	keyEpochUntrusted  = "restore.epochUntrusted"
)

func backup(t *testing.T, p *pair) []byte {
	t.Helper()
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return kept
}

func restoreServer(t *testing.T, p *pair, kept []byte) {
	t.Helper()
	if err := p.server.Fake.Restore(kept); err != nil {
		t.Fatal(err)
	}
}

func announce(t *testing.T, p *pair) []byte {
	t.Helper()
	id, err := p.server.Fake.AnnounceRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func readServerNotes(t *testing.T, p *pair) map[string]string {
	t.Helper()
	key := p.a.GetVaultKey(t, source)
	notes := map[string]string{}
	for _, head := range p.a.ReadChanges(t, p.a.GetRemoteID(t, source)) {
		if head.GetRevision().GetDeleted() {
			continue
		}
		path := synctest.OpenMeta(t, key, head.GetRevision()).Path
		notes[path] = string(p.a.OpenBlob(t, key, head.GetRevision()))
	}
	return notes
}

func wantServerNote(t *testing.T, p *pair, path, text string) {
	t.Helper()
	if got, ok := readServerNotes(t, p)[path]; !ok || got != text {
		t.Errorf("the server holds %q (there: %v) for %s, want %q", got, ok, path, text)
	}
}

func countCommits(p *pair) int {
	return p.server.Fake.CountCalls(syncv1connect.RevisionServiceCommitRevisionsProcedure)
}

func TestSyncVault_SendsANoteCreatedAfterTheBackupToTheRestoredServerWithItsOldFileID(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	before := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))[0].GetRevision()
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.a, source)

	wantServerNote(t, p, "Shopping list.md", "milk")
	after := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))[0].GetRevision()
	if !bytes.Equal(after.GetFileId(), before.GetFileId()) || len(after.GetParentRev()) != 0 {
		t.Errorf("revision = file %x parent %x, want the old file id and no parent", after.GetFileId(), after.GetParentRev())
	}
	if p.a.HasItemKey(keyRunning) {
		t.Error("the recovery is still reported after it ended")
	}
}

func TestSyncVault_SendsEachNoteOnlyOnceAfterTheRestore(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	announce(t, p)
	sync(t, p.a, source)
	commits := countCommits(p)

	sync(t, p.a, source)

	if got := countCommits(p); got != commits {
		t.Errorf("commits grew from %d to %d on the pass after the recovery, want none", commits, got)
	}
}

func TestSyncVault_KeepsTheNewestTextOnBothComputersWhenTheRestoredServerHasAnOlderOne(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "bread")
	sync(t, p.a, source)
	sync(t, p.b, target)
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.a, source)
	sync(t, p.b, target)

	wantNote(t, p.a, source, "Shopping list.md", "bread")
	wantNote(t, p.b, target, "Shopping list.md", "bread")
	wantServerNote(t, p, "Shopping list.md", "bread")
}

func TestSyncVault_TheSecondComputerThatSyncsFirstSendsItsTextOverTheOlderHead(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "bread")
	sync(t, p.a, source)
	sync(t, p.b, target)
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "bread")
	wantServerNote(t, p, "Shopping list.md", "bread")
}

func TestSyncVault_KeepsANoteRemovedAfterTheBackupRemoved(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	kept := backup(t, p)
	p.a.Vaults.With(source).Remove("Shopping list.md")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.a, source)
	sync(t, p.b, target)

	if _, ok := readServerNotes(t, p)["Shopping list.md"]; ok {
		t.Error("the server holds the removed note again")
	}
	if _, ok := p.a.ReadNote(source, "Shopping list.md"); ok {
		t.Error("the removed note is back on the first computer")
	}
	if _, ok := p.b.ReadNote(target, "Shopping list.md"); ok {
		t.Error("the removed note stays on the second computer")
	}
}

func TestSyncVault_SendsTheNoteOfAComputerThatWasAwayDuringTheRestoreOnce(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.b.GoOffline()
	p.b.WriteNote(target, "Shopping list.md", "eggs")
	restoreServer(t, p, kept)
	announce(t, p)
	p.b.GoOnline()
	commits := countCommits(p)

	sync(t, p.b, target)

	wantServerNote(t, p, "Shopping list.md", "eggs")
	if got := countCommits(p); got != commits+1 {
		t.Errorf("commits = %d, want %d: one commit for the note", got, commits+1)
	}
}

func TestSyncVault_ListsTheVaultConnectedAfterTheBackupAgain(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.AddVault(t, "Work")
	p.a.Connect(t, "Work")
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.a, source)

	listed, err := catalog.NewListRemoteVaults(p.b.CatalogDeps()).Execute(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("remote vaults = %+v, want Test and Work", listed)
	}
	names := map[string]bool{}
	for _, one := range listed {
		names[one.Name] = true
	}
	if !names["Work"] {
		t.Errorf("remote vaults = %+v, want Work among them", listed)
	}
}

func TestSyncVault_SendsTheChainOfDevicesTheRestoredServerLost(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	a := synctest.NewFirstComputer(t, server, clock, string(source), words)
	a.Connect(t, source)
	p := &pair{server: server, a: a}
	kept := backup(t, p)
	p.b = synctest.NewSecondComputer(t, server, clock, string(target), words)
	p.b.ConnectExisting(t, target, a.Link(t, source).RemoteVaultID)
	sync(t, p.b, target)
	restoreServer(t, p, kept)
	announce(t, p)

	sync(t, p.b, target)

	own, err := p.b.Account.GetPublicIdentity(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := p.a.Cloud.GetAccountChain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, own.Chain) {
		t.Error("the chain on the server is not the chain of the second computer after the recovery")
	}
}

func TestSyncVault_AppliesAndSendsNothingWhenTheServerWentBackWithoutAnnouncement(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "bread")
	sync(t, p.a, source)
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "eggs")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	commits := countCommits(p)

	sync(t, p.a, source)

	if !p.a.HasItem(textWentBack) {
		t.Errorf("health items = %+v, want %q", p.a.Health.List(), textWentBack)
	}
	wantNote(t, p.a, source, "Shopping list.md", "eggs")
	wantServerNote(t, p, "Shopping list.md", "bread")
	if got := countCommits(p); got != commits {
		t.Errorf("commits grew from %d to %d, want none while the server is not announced", commits, got)
	}
}

func TestSyncVault_ClearsTheWentBackWarningWhenTheRestoreIsAnnounced(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "bread")
	sync(t, p.a, source)
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "eggs")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	sync(t, p.a, source)
	announce(t, p)

	sync(t, p.a, source)

	if p.a.HasItemKey(keyWentBack + "/" + string(source)) {
		t.Errorf("health items = %+v, want the warning cleared", p.a.Health.List())
	}
	wantServerNote(t, p, "Shopping list.md", "eggs")
}

func TestSyncVault_FinishesARecoveryAfterAnInterruptedOne(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.WriteNote(source, "One.md", "milk")
	p.a.WriteNote(source, "Two.md", "tea")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	id := announce(t, p)
	p.server.LoseNextResponse("/CommitRevisions")

	err := p.a.Sync(t.Context(), source)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Fatalf("pass with the response lost = %v, want ErrUnreachable", err)
	}
	if got := p.a.Link(t, source).RestoreEpochID; bytes.Equal(got, id) {
		t.Error("the link holds the new epoch id before the recovery ended")
	}
	sync(t, p.a, source)
	wantServerNote(t, p, "One.md", "milk")
	wantServerNote(t, p, "Two.md", "tea")
	link := p.a.Link(t, source)
	if !bytes.Equal(link.RestoreEpochID, id) || link.Recovery != nil {
		t.Errorf("link = epoch %x recovery %+v, want the new epoch id and no recovery left", link.RestoreEpochID, link.Recovery)
	}
}

func TestSyncVault_KeepsTheNotesWhenTheCloudDeletedTheVaultAndSendsThemAfterANewSubscription(t *testing.T) {
	p := newSubscribedPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	setPhase(t, p.server, "deleted")
	if err := p.server.Fake.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	if err := p.a.Sync(t.Context(), source); err != nil && !errors.Is(err, syncprotocol.ErrNoVault) {
		t.Fatalf("pass over a deleted vault = %v", err)
	}

	wantNote(t, p.a, source, "Shopping list.md", "milk")
	setPhase(t, p.server, "active")
	sync(t, p.a, source)
	wantServerNote(t, p, "Shopping list.md", "milk")
	wantNote(t, p.a, source, "Shopping list.md", "milk")
}

func TestSyncVault_DeletesNoNoteWhenTheVaultIsUnknownWhileTheSubscriptionIsActive(t *testing.T) {
	p := newSubscribedPair(t)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	if err := p.server.Fake.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	sync(t, p.a, source)

	wantNote(t, p.a, source, "Shopping list.md", "milk")
	wantServerNote(t, p, "Shopping list.md", "milk")
}

func TestSyncVault_KeepsACachedCatalogLongerThanTheLogOfARolledBackServer(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.AddVault(t, "Work")
	p.a.Connect(t, "Work")
	cached := readCatalogCache(t, p.a)
	restoreServer(t, p, kept)
	serverEntries := len(p.a.ReadCatalog(t))

	if _, err := catalog.NewListRemoteVaults(p.a.CatalogDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}

	if got := readCatalogCache(t, p.a); len(got) != len(cached) || len(got) <= serverEntries {
		t.Errorf("cached catalog entries = %d, want the %d read before the server went back, more than the %d it holds", len(got), len(cached), serverEntries)
	}
}

func readCatalogCache(t *testing.T, c *synctest.Computer) [][]byte {
	t.Helper()
	journal, err := c.State.ReadJournal(t.Context(), restore.KindCatalog)
	if err != nil {
		t.Fatal(err)
	}
	return journal.Entries
}

func TestSyncVault_AppliesNewerTextInsteadOfSendingItsOwnWhenServerAnnouncesRestoreItNeverMade(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	sync(t, p.b, target)
	p.a.WriteNote(source, "Shopping list.md", "bread")
	sync(t, p.a, source)
	p.a.WriteNote(source, "Shopping list.md", "eggs")
	sync(t, p.a, source)
	announce(t, p)

	sync(t, p.b, target)

	wantNote(t, p.b, target, "Shopping list.md", "eggs")
	wantServerNote(t, p, "Shopping list.md", "eggs")
}

func TestSyncVault_IgnoresARestoreAnnouncementNoKnownRootVouchesForAndKeepsTheLocalNotes(t *testing.T) {
	p := newPair(t, fakeserver.Config{})
	kept := backup(t, p)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	sync(t, p.a, source)
	restoreServer(t, p, kept)
	announce(t, p)
	p.a.ClearRoots()
	commits := countCommits(p)

	sync(t, p.a, source)

	if !p.a.HasItem(textEpochUntrusted) || !p.a.HasItemKey(keyEpochUntrusted+"/"+string(source)) {
		t.Errorf("health items = %+v, want %q", p.a.Health.List(), textEpochUntrusted)
	}
	if got := countCommits(p); got != commits {
		t.Errorf("commits grew from %d to %d, want none for an announcement nobody vouches for", commits, got)
	}
	if _, ok := readServerNotes(t, p)["Shopping list.md"]; ok {
		t.Error("the note reached the restored server on the word of an announcement nobody vouches for")
	}
	wantNote(t, p.a, source, "Shopping list.md", "milk")
}
