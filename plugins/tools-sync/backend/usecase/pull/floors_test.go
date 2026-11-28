package pull_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

func (p *pair) fillFloors(t *testing.T, count int) {
	t.Helper()
	err := p.b.State.UpdateLink(t.Context(), target, func(link *vaultlink.Link) error {
		for n := range count {
			id := make([]byte, 16)
			binary.BigEndian.PutUint64(id, uint64(n)+1)
			link.AddRetired(vaultlink.CreateRetiredFile(id, 2, p.b.Clock.Now()))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPullChanges_KeepsTheRemovalOfAFileWhenTenThousandMoreFilesWereRemovedAfterIt(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Draft.md", "draft", nil)
	older := p.a.ReadChanges(t, p.vault.ID)
	p.pull(t, trustAll)
	p.a.CommitRemoval(t, p.vault, "Draft.md", first)
	p.pull(t, trustAll)
	p.fillFloors(t, 10001)

	p.pullServing(t, older...)

	if _, isThere := p.b.ReadNote(target, "Draft.md"); isThere {
		t.Error("the removed note came back from an old revision after thousands of later removals")
	}
	if !p.b.HasItemKey(pull.KeyRollback) {
		t.Error("no health item names the old revision of the removed note")
	}
}

func TestPullChanges_KeepsTheRemovalUntilTheEndOfTheRetentionOfItsFloor(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Draft.md", "draft", nil)
	older := p.a.ReadChanges(t, p.vault.ID)
	p.pull(t, trustAll)
	p.a.CommitRemoval(t, p.vault, "Draft.md", first)
	p.pull(t, trustAll)
	p.b.Clock.Advance(vaultlink.FloorRetention - time.Hour)
	p.pull(t, trustAll)

	p.pullServing(t, older...)

	if _, isThere := p.b.ReadNote(target, "Draft.md"); isThere {
		t.Error("the removed note came back before the retention of its floor ended")
	}
	if !p.b.HasItemKey(pull.KeyRollback) {
		t.Error("no health item names the old revision of the removed note")
	}
}

func TestPullChanges_ForgetsTheFloorOfARemovalAfterTheRetentionEnds(t *testing.T) {
	p := newPair(t)
	first := p.push(t, "Draft.md", "draft", nil)
	p.pull(t, trustAll)
	p.a.CommitRemoval(t, p.vault, "Draft.md", first)
	p.pull(t, trustAll)
	p.b.Clock.Advance(vaultlink.FloorRetention + time.Hour)

	p.pull(t, trustAll)

	link := p.b.Link(t, target)
	if got := link.GetFloor(first.FileID); got != 0 {
		t.Errorf("floor = %d after the retention, want it dropped", got)
	}
}
