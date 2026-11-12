package indexing

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// readyLog records the vaults a worker told were embedded.
type readyLog struct {
	vaults []domain.VaultID
}

func (l *readyLog) report(_ context.Context, v domain.Vault) {
	l.vaults = append(l.vaults, v.ID)
}

func readyPass(t *testing.T, openReading openFunc, model searchsourceport.Embedder, log *readyLog) *testPass {
	t.Helper()
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, newRecorder(), model, model.Describe().Recipe(), v)
	p.worker.searchSourceReady = log.report
	return p
}

func testAPassThatEmbeddedEverythingTellsThatTheVectorsOfItsVaultAreReady(t *testing.T, openReading openFunc) {
	t.Helper()
	log := &readyLog{}
	p := readyPass(t, openReading, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())

	if len(log.vaults) != 1 || log.vaults[0] != p.v.ID {
		t.Fatalf("told %v, want the vault of the pass once", log.vaults)
	}
}

func testAPassWithNothingPendingTellsAgain(t *testing.T, openReading openFunc) {
	t.Helper()
	log := &readyLog{}
	p := readyPass(t, openReading, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())
	p.embed(t.Context())

	if len(log.vaults) != 2 {
		t.Errorf("the worker told %d times, want once for each pass", len(log.vaults))
	}
}

func testAPassThatCouldNotEmbedTellsNothing(t *testing.T, openReading openFunc) {
	t.Helper()
	log := &readyLog{}
	p := readyPass(t, openReading, &sulking{fakeEmbedder: &fakeEmbedder{dims: 64}}, log)

	p.embed(t.Context())

	if len(log.vaults) != 0 {
		t.Errorf("told %v after a pass that could not embed", log.vaults)
	}
}

func testAPassThatWasStoppedTellsNothing(t *testing.T, openReading openFunc) {
	t.Helper()
	log := &readyLog{}
	p := readyPass(t, openReading, &fakeEmbedder{dims: 64}, log)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p.embed(ctx)

	if len(log.vaults) != 0 {
		t.Errorf("told %v in a pass that was stopped", log.vaults)
	}
}

func testAWorkerWithoutAListenerEmbedsAsBefore(t *testing.T, openReading openFunc) {
	t.Helper()
	made, db := openReading(t)
	model := &fakeEmbedder{dims: 64}
	fit(t, db, model)
	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.Describe().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, work.MakingVectors); pos != nil {
		t.Errorf("a pass without a listener is still being done: %+v", *pos)
	}
}
