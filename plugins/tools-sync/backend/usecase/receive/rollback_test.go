package receive_test

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/receive"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

// olderLog is a server that serves the share log as it was: only the entries up to a version, and the owner chain it
// had.
type olderLog struct {
	ports.Shares
	last  uint64
	chain []byte
	fork  bool
}

func (o olderLog) GetShareLog(ctx context.Context, id []byte, since uint64) ([]*syncv1.ShareLogEntry, []byte, error) {
	entries, chain, err := o.Shares.GetShareLog(ctx, id, since)
	var kept []*syncv1.ShareLogEntry
	for _, entry := range entries {
		if entry.GetVersion() > o.last {
			continue
		}
		if o.fork {
			entry.EncShareBody = append([]byte{0}, entry.GetEncShareBody()...)
		}
		kept = append(kept, entry)
	}
	if o.chain != nil {
		chain = o.chain
	}
	return kept, chain, err
}

// olderChanges is a server that answers every read of the share changes with a page it served before.
type olderChanges struct {
	ports.Shares
	page *syncv1.GetShareChangesResponse
}

func (o olderChanges) GetShareChanges(_ context.Context, _ []byte, _ uint64, _ uint32) (*syncv1.GetShareChangesResponse, error) {
	return o.page, nil
}

func (w *world) shareID(t *testing.T, id domain.VaultID) []byte {
	t.Helper()
	raw, err := hex.DecodeString(w.reader.Link(t, id).Received.ShareRootID)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (w *world) pullWith(t *testing.T, id domain.VaultID, shares ports.Shares) error {
	t.Helper()
	deps := w.reader.ReceiveDeps()
	deps.Shares = shares
	_, err := receive.NewPullShare(deps).Execute(t.Context(), id)
	return err
}

func TestPullShare_RejectsLogMissingReaderLastEntry(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	w.shareWith(t, synctest.NewReaderComputer(t, w.server, w.clock, "Bo"), "Bo")
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}
	held := w.reader.Link(t, received.VaultID).Received.ServerVersion

	err := w.pullWith(t, received.VaultID, olderLog{Shares: w.reader.Cloud, last: held - 1, chain: nil, fork: false})

	if !errors.Is(err, share.ErrRolledBack) {
		t.Errorf("error = %v, want ErrRolledBack for a log that ends before the entry the reader holds", err)
	}
	if got := w.reader.Link(t, received.VaultID).Received.ServerVersion; got != held {
		t.Errorf("the reader holds version %d after the refused pass, want %d", got, held)
	}
}

func TestPullShare_RejectsLogWithDifferentEntryAtReaderPosition(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	held := w.reader.Link(t, received.VaultID).Received.ServerVersion

	err := w.pullWith(t, received.VaultID, olderLog{Shares: w.reader.Cloud, last: held, chain: nil, fork: true})

	if !errors.Is(err, share.ErrRolledBack) {
		t.Errorf("error = %v, want ErrRolledBack for an entry whose id differs from the stored one", err)
	}
}

func TestPullShare_ErrorsOnRevisionOfEpochAboveKnownLog(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	removed := w.shareWith(t, w.reader, "Anna")
	staying := synctest.NewReaderComputer(t, w.server, w.clock, "Bo")
	stayingVault := w.receive(t, staying, w.shareWith(t, staying, "Bo"))
	if err := w.owner.RemoveReader(t.Context(), owned, folder, removed.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	held := staying.Link(t, stayingVault.VaultID).Received.ServerVersion
	deps := staying.ReceiveDeps()
	deps.Shares = olderLog{Shares: staying.Cloud, last: held, chain: nil, fork: false}

	_, err := receive.NewPullShare(deps).Execute(t.Context(), stayingVault.VaultID)

	if !errors.Is(err, share.ErrRolledBack) {
		t.Errorf("error = %v, want ErrRolledBack for a revision of epoch 1 while the log held ends at epoch 0", err)
	}
	wantNoNote(t, staying, stayingVault.VaultID, "Bread.md")
	if err := staying.Sync(t.Context(), stayingVault.VaultID); err != nil {
		t.Fatal(err)
	}
	wantNote(t, staying, stayingVault.VaultID, "Bread.md", "flour")
}

func TestPullShare_RefusesAFileServedAtAnOlderRevisionThanTheReaderSaw(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	older, err := w.reader.Cloud.GetShareChanges(t.Context(), w.shareID(t, received.VaultID), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots and leeks")
	w.syncOwner(t)
	if err := w.reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatal(err)
	}

	err = w.pullWith(t, received.VaultID, olderChanges{Shares: w.reader.Cloud, page: older})

	if !errors.Is(err, share.ErrRolledBack) {
		t.Errorf("error = %v, want ErrRolledBack for a file served at its first revision again", err)
	}
	wantNote(t, w.reader, received.VaultID, "Soup.md", "carrots and leeks")
}

func TestPullShare_AcceptsTheSameRevisionServedAgain(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	page, err := w.reader.Cloud.GetShareChanges(t.Context(), w.shareID(t, received.VaultID), 0, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.pullWith(t, received.VaultID, olderChanges{Shares: w.reader.Cloud, page: page}); err != nil {
		t.Errorf("a head served twice is no rollback: %v", err)
	}
}

func TestPullShare_RefusesAnOwnerChainOlderThanTheOneStored(t *testing.T) {
	words := synctest.NewWords(t)
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	first := synctest.NewFirstComputer(t, server, clock, string(owned), words)
	first.Connect(t, owned)
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	w := &world{server: server, clock: clock, owner: first, reader: reader}
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, reader, w.shareWith(t, reader, "Anna"))
	_, before, err := reader.Cloud.GetShareLog(t.Context(), w.shareID(t, received.VaultID), 0)
	if err != nil {
		t.Fatal(err)
	}
	synctest.NewSecondComputer(t, server, clock, "Copy", words)
	if err := reader.Sync(t.Context(), received.VaultID); err != nil {
		t.Fatalf("a chain that extends the stored one is accepted: %v", err)
	}
	held := reader.Link(t, received.VaultID).Received.ServerVersion
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)

	err = w.pullWith(t, received.VaultID, olderLog{Shares: reader.Cloud, last: held, chain: before, fork: false})

	if err == nil {
		t.Error("a pass over an owner chain shorter than the stored one succeeded, want a refusal")
	}
	wantNoNote(t, reader, received.VaultID, "Bread.md")
}

// renamingShares renames the vault in the link while the pull reads the changes of the share.
type renamingShares struct {
	ports.Shares
	rename func()
}

func (r renamingShares) GetShareChanges(ctx context.Context, id []byte, since uint64, limit uint32) (*syncv1.GetShareChangesResponse, error) {
	r.rename()
	return r.Shares.GetShareChanges(ctx, id, since, limit)
}

func TestPullShare_KeepsAChangeOfTheLinkMadeWhileItRuns(t *testing.T) {
	w := newWorld(t)
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, w.reader, w.shareWith(t, w.reader, "Anna"))
	w.owner.WriteNote(owned, "Food/Stew.md", "beef")
	w.syncOwner(t)
	shares := renamingShares{Shares: w.reader.Cloud, rename: func() {
		err := w.reader.State.UpdateLink(t.Context(), received.VaultID, func(link *vaultlink.Link) error {
			link.CatalogName = "Renamed"
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	}}

	if err := w.pullWith(t, received.VaultID, shares); err != nil {
		t.Fatal(err)
	}

	if got := w.reader.Link(t, received.VaultID).CatalogName; got != "Renamed" {
		t.Errorf("catalog name = %q, want Renamed", got)
	}
	wantNote(t, w.reader, received.VaultID, "Stew.md", "beef")
}

func TestAcceptInvitation_RefusesAnOwnerChainShorterThanTheOneOfTheInvitation(t *testing.T) {
	words := synctest.NewWords(t)
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	first := synctest.NewFirstComputer(t, server, clock, string(owned), words)
	first.Connect(t, owned)
	reader := synctest.NewReaderComputer(t, server, clock, "Reader")
	w := &world{server: server, clock: clock, owner: first, reader: reader}
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	received := w.receive(t, reader, w.shareWith(t, reader, "Anna"))
	_, before, err := reader.Cloud.GetShareLog(t.Context(), w.shareID(t, received.VaultID), 0)
	if err != nil {
		t.Fatal(err)
	}
	synctest.NewSecondComputer(t, server, clock, "Copy", words)
	second := synctest.NewReaderComputer(t, server, clock, "Second")
	invitation := w.shareWith(t, second, "Bo")
	deps := second.ReceiveDeps()
	deps.Shares = olderLog{Shares: second.Cloud, last: math.MaxUint64, chain: before, fork: false}

	_, err = receive.NewAcceptInvitation(deps).Execute(t.Context(), invitation.Text)

	if err == nil {
		t.Error("an invitation was accepted over an owner chain shorter than its own, want a refusal")
	}
}
