package restore_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/restore"
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
	server := synctest.NewServer(t, clock)
	words := synctest.NewWords(t)
	return &pair{server: server, a: synctest.NewFirstComputer(t, server, clock, source, words), words: words, clock: clock}
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

func TestRecoverAccount_RefusesAServerThatAnnouncedNoRestore(t *testing.T) {
	t.Parallel()
	p := newFirst(t)
	p.a.Connect(t, source)

	err := usecase.NewRecoverAccount(p.a.RestoreDeps()).Execute(t.Context())

	if !errors.Is(err, usecase.ErrEpochUntrusted) {
		t.Errorf("Execute = %v, want ErrEpochUntrusted", err)
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
