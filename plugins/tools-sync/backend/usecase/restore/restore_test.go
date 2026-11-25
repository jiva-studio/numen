package restore_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/restore"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const (
	source = "Test"
	target = "Copy"
)

type pair struct {
	server *synctest.Server
	a      *synctest.Computer
	words  []string
	clock  *synctest.Clock
}

func newFirst(t *testing.T) *pair {
	t.Helper()
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	return &pair{server: server, a: synctest.NewFirstComputer(t, server, clock, source, words), words: words, clock: clock}
}

func (p *pair) restoreAndAnnounce(t *testing.T, kept []byte) {
	t.Helper()
	if err := p.server.Fake.Restore(kept); err != nil {
		t.Fatal(err)
	}
	if _, err := p.server.Fake.AnnounceRestoreEpoch(); err != nil {
		t.Fatal(err)
	}
}

func readCache(t *testing.T, c *synctest.Computer) []*syncv1.CatalogEntry {
	t.Helper()
	journal, err := c.State.ReadJournal(t.Context(), restore.KindCatalog)
	if err != nil {
		t.Fatal(err)
	}
	var entries []*syncv1.CatalogEntry
	for _, raw := range journal.Entries {
		entry, decodeErr := syncprotocol.DecodeCatalogEntry(raw)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestCatalogCache_ContainsTheVerifiedEntriesInOrderAfterAConnectAndARename(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	if _, err := p.a.Vaults.RenameVault(t.Context(), source, "Work"); err != nil {
		t.Fatal(err)
	}
	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	cached := readCache(t, p.a)

	server := p.a.ReadCatalog(t)
	if len(cached) != 2 || len(server) != 2 {
		t.Fatalf("cache holds %d entries, server %d, want 2 each", len(cached), len(server))
	}
	for i := range cached {
		if !isSameEntry(t, cached[i], server[i]) {
			t.Errorf("cached entry %d differs from the entry on the server", i)
		}
	}
}

func TestRecoverAccount_SendsTheCachedCatalogEntriesTheServerLost(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	p.a.Connect(t, source)
	p.restoreAndAnnounce(t, kept)

	err = usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	cached, server := readCache(t, p.a), p.a.ReadCatalog(t)
	if len(server) != 1 || len(cached) != 1 || !isSameEntry(t, cached[0], server[0]) {
		t.Errorf("server holds %d entries, want the one entry of the cache again", len(server))
	}
}

func TestRecoverAccount_RefusesAnAnnouncementNoKnownRootVouchesForAndSendsNothing(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	p.a.Connect(t, source)
	p.restoreAndAnnounce(t, kept)
	p.a.ClearRoots()

	err = usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if !errors.Is(err, usecase.ErrEpochUntrusted) {
		t.Errorf("Execute = %v, want ErrEpochUntrusted", err)
	}
	if got := len(p.a.ReadCatalog(t)); got != 0 {
		t.Errorf("server holds %d catalog entries, want none sent", got)
	}
}

func TestRecoverAccount_RefusesAServerThatAnnouncedNoRestore(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)

	err := usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if !errors.Is(err, usecase.ErrEpochUntrusted) {
		t.Errorf("Execute = %v, want ErrEpochUntrusted", err)
	}
}

func TestRecoverAccount_SendsTheCatalogAgainToAServerThatLostTheAccountDataAndAnnouncedNoRestore(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	if err := p.server.Fake.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	err := usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if err != nil {
		t.Fatal(err)
	}
	if got := len(p.a.ReadCatalog(t)); got != 1 {
		t.Errorf("server holds %d catalog entries, want the one of the cache again", got)
	}
}

func TestRecoverAccount_StopsWhenTheLogOfTheServerDiffersFromTheCache(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	p.a.Connect(t, source)
	p.restoreAndAnnounce(t, kept)
	other := synctest.NewSecondComputer(t, p.server, p.clock, target, p.words)
	other.Connect(t, target)

	err = usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if !errors.Is(err, usecase.ErrDiverged) {
		t.Errorf("Execute = %v, want ErrDiverged", err)
	}
	if got := len(p.a.ReadCatalog(t)); got != 1 {
		t.Errorf("server holds %d catalog entries, want the one of the other computer untouched", got)
	}
}

func TestRecoverVault_SendsNothingForAServerThatHasTheHeadsThisDeviceRecorded(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	p.a.WriteNote(source, "Shopping list.md", "milk")
	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	heads := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))

	if err := usecase.NewRecoverVault(p.a.RestoreDeps()).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	after := p.a.ReadChanges(t, p.a.GetRemoteID(t, source))
	if len(after) != 1 || !bytes.Equal(after[0].GetRevId(), heads[0].GetRevId()) {
		t.Errorf("heads after the recovery = %v, want %v unchanged", after, heads)
	}
}

func TestRecoverVault_WritesTheEpochIDToTheLinkOnlyAfterTheLastNote(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	p.a.WriteNote(source, "Shopping list.md", "milk")
	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	p.restoreAndAnnounce(t, kept)
	p.server.LoseNextResponse("/CommitRevisions")

	err = usecase.NewRecoverVault(p.a.RestoreDeps()).Execute(t.Context(), source)

	if err == nil {
		t.Fatal("recovery with the response of the commit lost succeeded, want an error")
	}
	link := p.a.Link(t, source)
	if len(link.RestoreEpochID) != 0 || link.Recovery == nil {
		t.Errorf("link = epoch %x recovery %+v, want no epoch id and the progress kept", link.RestoreEpochID, link.Recovery)
	}
}

func isSameEntry(t *testing.T, a, b *syncv1.CatalogEntry) bool {
	t.Helper()
	left, err := syncprotocol.EncodeCatalogEntry(a)
	if err != nil {
		t.Fatal(err)
	}
	right, err := syncprotocol.EncodeCatalogEntry(b)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Equal(left, right)
}

type renamingCommitter struct {
	usecase.Committer
	rename func()
}

func (r renamingCommitter) Execute(ctx context.Context, in ports.CommitInput) ([]transfer.Outcome, error) {
	r.rename()
	return r.Committer.Execute(ctx, in)
}

func TestRecoverVault_KeepsAChangeOfTheLinkMadeWhileItRuns(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	kept, err := p.server.Fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	p.a.WriteNote(source, "Shopping list.md", "milk")
	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	p.restoreAndAnnounce(t, kept)
	deps := p.a.RestoreDeps()
	deps.Commit = renamingCommitter{Committer: deps.Commit, rename: func() {
		err := p.a.State.UpdateLink(t.Context(), source, func(link *vaultlink.Link) error {
			link.CatalogName = "Renamed"
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}}

	if err := usecase.NewRecoverVault(deps).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	link := p.a.Link(t, source)
	if link.CatalogName != "Renamed" || len(link.RestoreEpochID) == 0 {
		t.Errorf("link = name %q epoch %x, want the name Renamed and the epoch id of the recovery", link.CatalogName, link.RestoreEpochID)
	}
}

func TestRecoverAccount_DropsTheCachedCatalogOfAnotherIdentityAndSendsNothing(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	clock := synctest.NewClock(synctest.GetStart())
	other := synctest.NewFirstComputer(t, synctest.NewServer(t, clock, fakeserver.Config{}), clock, target, synctest.NewWords(t))
	stale, err := syncprotocol.EncodeCatalogEntry(other.NewCatalogEntry(t, nil, bytes.Repeat([]byte{7}, 16), true, []byte("body")))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.a.State.WriteJournal(t.Context(), restore.Journal{Kind: restore.KindCatalog, Entries: [][]byte{stale}}); err != nil {
		t.Fatal(err)
	}
	if err := p.server.Fake.DeleteAccountData(); err != nil {
		t.Fatal(err)
	}

	err = usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if !errors.Is(err, usecase.ErrForeignJournal) {
		t.Errorf("Execute = %v, want ErrForeignJournal", err)
	}
	if got := len(readCache(t, p.a)); got != 0 {
		t.Errorf("cache holds %d entries, want it dropped", got)
	}
	if got := len(p.a.ReadCatalog(t)); got != 0 {
		t.Errorf("server holds %d catalog entries, want none sent", got)
	}
}

func TestRecoverVault_SkipsRevisionWithBadSignature(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	other := synctest.NewSecondComputer(t, p.server, p.clock, target, p.words)
	other.ConnectExisting(t, target, p.a.Link(t, source).RemoteVaultID)
	other.WriteNote(target, "Plan.md", "tea")
	if err := other.Sync(t.Context(), target); err != nil {
		t.Fatal(err)
	}
	forged := p.a.RestoreDeps()
	forged.Revisions = synctest.ForgedSignatures{Revisions: forged.Revisions}

	if err := usecase.NewRecoverVault(forged).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	if text, isThere := p.a.ReadNote(source, "Plan.md"); isThere {
		t.Fatalf("note = %q, want none: its revision has a bad signature", text)
	}

	if err := usecase.NewRecoverVault(p.a.RestoreDeps()).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	if err := p.a.Sync(t.Context(), source); err != nil {
		t.Fatal(err)
	}
	if text, isThere := p.a.ReadNote(source, "Plan.md"); !isThere || text != "tea" {
		t.Errorf("note = %q (there: %v), want tea from the honest server", text, isThere)
	}
}

func TestRecoverVault_RefusesAnEpochIdThatTheSignedAnnouncementDoesNotName(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	if _, err := p.server.Fake.AnnounceRestoreEpoch(); err != nil {
		t.Fatal(err)
	}
	claimed := p.a.RestoreDeps()
	claimed.Revisions = synctest.EpochClaim{Revisions: claimed.Revisions, ID: bytes.Repeat([]byte{0xAB}, 16)}

	err := usecase.NewRecoverVault(claimed).Execute(t.Context(), source)

	if !errors.Is(err, usecase.ErrEpochUntrusted) {
		t.Errorf("Execute = %v, want ErrEpochUntrusted", err)
	}
	if got := p.a.Link(t, source).RestoreEpochID; len(got) != 0 {
		t.Errorf("epoch id of the link = %x, want none stored from an unsigned claim", got)
	}
}

func TestRecoverVault_StoresTheEpochIdOfTheSignedAnnouncement(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)
	announced, err := p.server.Fake.AnnounceRestoreEpoch()
	if err != nil {
		t.Fatal(err)
	}

	if err := usecase.NewRecoverVault(p.a.RestoreDeps()).Execute(t.Context(), source); err != nil {
		t.Fatal(err)
	}

	if got := p.a.Link(t, source).RestoreEpochID; !bytes.Equal(got, announced) {
		t.Errorf("epoch id of the link = %x, want %x of the announcement", got, announced)
	}
}
