package syncvault_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/deletions"
)

func (p *pair) pauseRemovals(t *testing.T, count int) {
	t.Helper()
	p.removeMany(count)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	if pending, err := p.a.State.ReadPending(t.Context(), source); err != nil || pending == nil {
		t.Fatalf("pending = %+v (%v), want a pause of %d removals", pending, err, count)
	}
}

func (p *pair) answer(t *testing.T, choice transfer.Choice) {
	t.Helper()
	if _, err := deletions.NewResolveDeletions(p.a.DeleteDeps()).Execute(t.Context(), source, choice); err != nil {
		t.Fatal(err)
	}
}

func (p *pair) wantPaused(t *testing.T, count int) {
	t.Helper()
	pending, err := p.a.State.ReadPending(t.Context(), source)
	if err != nil || pending == nil || len(pending.Files) != count {
		t.Fatalf("pending = %+v (%v), want a new pause of %d removals", pending, err, count)
	}
	if live := p.getLiveHeads(t); len(live) != 20 {
		t.Errorf("live files on the server = %d, want all 20: nothing was sent", len(live))
	}
}

func TestSyncVault_AnAnswerIsDroppedWhenThePersonRestoredTheFilesBeforeTheNextPass(t *testing.T) {
	p := newPair(t)
	p.writeMany(t, 20)
	sync(t, p.a, source)
	p.pauseRemovals(t, 8)
	p.writeMany(t, 8)
	p.answer(t, transfer.ChoiceRemove)

	sync(t, p.a, source)

	if approval, err := p.a.State.ReadApproval(t.Context(), source); err != nil || approval != nil {
		t.Errorf("approval = %+v (%v), want none once a pass found nothing to remove", approval, err)
	}
	p.removeMany(8)
	sync(t, p.a, source)
	p.afterDelay()
	sync(t, p.a, source)
	p.wantPaused(t, 8)
}

func TestSyncVault_AnAnswerIsNotUsedForRemovalsOfALaterDay(t *testing.T) {
	p := newPair(t)
	p.writeMany(t, 20)
	sync(t, p.a, source)
	p.pauseRemovals(t, 8)
	p.answer(t, transfer.ChoiceRemove)
	p.a.Clock.Advance(deletions.ApprovalLifetime + time.Hour)

	sync(t, p.a, source)

	p.wantPaused(t, 8)
}

func TestSyncVault_AnAnswerForEightFilesDoesNotRemoveThirteen(t *testing.T) {
	p := newPair(t)
	p.writeMany(t, 20)
	sync(t, p.a, source)
	p.removeMany(8)
	sync(t, p.a, source)
	p.afterDelay()
	for i := 8; i < 13; i++ {
		p.a.Vaults.With(source).Remove(fmt.Sprintf("Papers/File %d.pdf", i+1))
	}
	sync(t, p.a, source)
	p.answer(t, transfer.ChoiceRemove)
	p.afterDelay()

	sync(t, p.a, source)

	p.wantPaused(t, 13)
}
