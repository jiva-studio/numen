package share_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	domainshare "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/commit"
	usecase "github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/share"
)

// staleShares answers the first read of a pass with the shares as they were, and the later reads as they are.
type staleShares struct {
	stale   []domainshare.Root
	current *usecase.ReadShares
	calls   int
}

func (s *staleShares) Execute(ctx context.Context, id domain.VaultID) ([]domainshare.Root, error) {
	s.calls++
	if s.calls == 1 {
		return s.stale, nil
	}
	return s.current.Execute(ctx, id)
}

func (w *world) removeAfterReading(t *testing.T) []domainshare.Root {
	t.Helper()
	w.shareFolder(t)
	first := w.addReader(t, w.reader, "Anna")
	stale, err := usecase.NewReadShares(w.owner.ShareDeps()).Execute(t.Context(), owned)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.owner.RemoveReader(t.Context(), owned, folder, first.ReaderFingerprint); err != nil {
		t.Fatal(err)
	}
	return stale
}

func TestCommitRevisions_ReturnsStaleEpochForNoteSealedAtOldEpoch(t *testing.T) {
	w := newWorld(t)
	stale := w.removeAfterReading(t)
	in := transfer.CommitInput{
		Vault:  transfer.Vault{ID: w.owner.GetRemoteID(t, owned), Key: w.owner.GetVaultKey(t, owned)},
		Files:  []transfer.File{{Path: "Food/Bread.md", Content: []byte("flour")}},
		Shares: stale,
	}

	outcomes, err := commit.NewCommitRevisions(w.owner.CommitDeps()).Execute(t.Context(), in)

	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].Status != transfer.StaleEpoch {
		t.Errorf("outcomes = %+v, want one note refused for the epoch of its share", outcomes)
	}
}

func TestPassSealedAtOldEpochReloadsSharesAndCommitsAtNewEpoch(t *testing.T) {
	w := newWorld(t)
	stale := w.removeAfterReading(t)
	w.owner.WriteNote(owned, "Food/Bread.md", "flour")
	reading := &staleShares{stale: stale, current: usecase.NewReadShares(w.owner.ShareDeps()), calls: 0}

	if err := w.owner.NewSyncVaultReading(reading).Execute(t.Context(), owned); err != nil {
		t.Fatal(err)
	}

	var epochs []uint32
	for _, head := range w.heads(t) {
		if len(head.GetRevision().GetShareRootId()) == 16 {
			epochs = append(epochs, head.GetRevision().GetEpoch())
		}
	}
	if len(epochs) != 1 || epochs[0] != 1 {
		t.Errorf("epochs of the share revisions = %v, want [1]", epochs)
	}
	if reading.calls != 2 {
		t.Errorf("the shares were read %d times, want 2: at the start and after the refusal", reading.calls)
	}
}
