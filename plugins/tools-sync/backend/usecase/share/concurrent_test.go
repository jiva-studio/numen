package share_test

import (
	"context"
	"testing"

	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

// racingShares is a server that lets another device of the owner append first: the call of the entry runs rival,
// once, before the entry reaches the server.
type racingShares struct {
	ports.Shares
	rival  func()
	isDone bool
}

func (r *racingShares) AppendShareLogEntry(ctx context.Context, entry *syncv1.ShareLogEntry) (*syncv1.AppendShareLogEntryResponse, error) {
	if !r.isDone {
		r.isDone = true
		r.rival()
	}
	return r.Shares.AppendShareLogEntry(ctx, entry)
}

func TestRemoveReader_LosingDeviceAdoptsWinnerKeyAndSealsNewKey(t *testing.T) {
	words := synctest.NewWords(t)
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	first := synctest.NewFirstComputer(t, server, clock, string(owned), words)
	first.Connect(t, owned)
	second := synctest.NewSecondComputer(t, server, clock, "Copy", words)
	second.ConnectExisting(t, "Copy", first.Link(t, owned).RemoteVaultID)
	w := &world{server: server, clock: clock, owner: first, reader: synctest.NewReaderComputer(t, server, clock, "Reader")}
	w.owner.WriteNote(owned, "Food/Soup.md", "carrots")
	w.syncOwner(t)
	w.shareFolder(t)
	removedA := w.addReader(t, w.reader, "Anna")
	other := synctest.NewReaderComputer(t, server, clock, "Bo")
	removedB := w.addReader(t, other, "Bo")
	staying := synctest.NewReaderComputer(t, server, clock, "Cy")
	stayingVault, err := staying.AcceptInvitation(t.Context(), w.addReader(t, staying, "Cy").Text)
	if err != nil {
		t.Fatal(err)
	}
	var winnerKey [32]byte
	rival := func() {
		if err := second.RemoveReader(t.Context(), "Copy", folder, removedB.ReaderFingerprint); err != nil {
			t.Error(err)
		}
		winnerKey = readEpochKey(t, w, 1)
	}
	deps := first.ShareDeps()
	deps.Shares = &racingShares{Shares: deps.Shares, rival: rival, isDone: false}

	err = usecase.NewRemoveReader(deps).Execute(t.Context(), owned, folder, removedA.ReaderFingerprint)

	if err != nil {
		t.Fatal(err)
	}
	roots, err := usecase.NewReadShares(first.ShareDeps()).Execute(t.Context(), owned)
	if err != nil || len(roots) != 1 {
		t.Fatalf("roots = %v, error = %v, want the one share", roots, err)
	}
	if roots[0].Epoch != 2 {
		t.Errorf("epoch = %d, want 2: the winner started epoch 1 and the loser epoch 2 on top of it", roots[0].Epoch)
	}
	if kept, isKnown := roots[0].Keys.Get(1); !isKnown || kept != winnerKey {
		t.Error("the key of epoch 1 in the head entry is not the key of the winner")
	}
	if fresh, isKnown := roots[0].Keys.Get(2); !isKnown || fresh == winnerKey {
		t.Error("the key of epoch 2 is missing or equals the key of epoch 1")
	}
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	w.syncOwner(t)
	if err := staying.Sync(t.Context(), stayingVault.VaultID); err != nil {
		t.Fatal(err)
	}
	if got, isThere := staying.ReadNote(stayingVault.VaultID, "Bread.md"); !isThere || got != "flour" {
		t.Errorf("the reader that stays reads %q (there: %v), want flour sealed under the key of epoch 2", got, isThere)
	}
}

func readEpochKey(t *testing.T, w *world, epoch uint32) [32]byte {
	t.Helper()
	roots, err := usecase.NewReadShares(w.owner.ShareDeps()).Execute(t.Context(), owned)
	if err != nil || len(roots) != 1 {
		t.Fatalf("roots = %v, error = %v", roots, err)
	}
	key, isKnown := roots[0].Keys.Get(epoch)
	if !isKnown {
		t.Fatalf("the share has no key for epoch %d", epoch)
	}
	return key
}
