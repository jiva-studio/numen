package indexing

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/readingfixture"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// readyLog records the vaults a worker told were embedded.
type readyLog struct {
	vaults []domain.VaultID
}

func (l *readyLog) report(_ context.Context, v domain.Vault) {
	l.vaults = append(l.vaults, v.ID)
}

func readyPass(t *testing.T, model embeddingport.Embedder, log *readyLog) *pass {
	t.Helper()
	made, db := readingfixture.Open(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(getBefore(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, newRecorder(), model, model.GetModel().Recipe(), v)
	p.worker.vectorsReady = log.report
	return p
}

func TestAPassThatEmbeddedEverythingTellsThatTheVectorsOfItsVaultAreReady(t *testing.T) {
	log := &readyLog{}
	p := readyPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())

	if len(log.vaults) != 1 || log.vaults[0] != p.v.ID {
		t.Fatalf("told %v, want the vault of the pass once", log.vaults)
	}
}

func TestAPassWithNothingPendingTellsAgain(t *testing.T) {
	log := &readyLog{}
	p := readyPass(t, &fakeEmbedder{dims: 64}, log)

	p.embed(t.Context())
	p.embed(t.Context())

	if len(log.vaults) != 2 {
		t.Errorf("the worker told %d times, want once for each pass", len(log.vaults))
	}
}

func TestAPassThatCouldNotEmbedTellsNothing(t *testing.T) {
	log := &readyLog{}
	p := readyPass(t, &sulking{fakeEmbedder: &fakeEmbedder{dims: 64}}, log)

	p.embed(t.Context())

	if len(log.vaults) != 0 {
		t.Errorf("told %v after a pass that could not embed", log.vaults)
	}
}

func TestAPassThatWasStoppedTellsNothing(t *testing.T) {
	log := &readyLog{}
	p := readyPass(t, &fakeEmbedder{dims: 64}, log)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p.embed(ctx)

	if len(log.vaults) != 0 {
		t.Errorf("told %v in a pass that was stopped", log.vaults)
	}
}

func TestAWorkerWithoutAListenerEmbedsAsBefore(t *testing.T) {
	made, db := readingfixture.Open(t)
	model := &fakeEmbedder{dims: 64}
	fit(t, db, model)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(getBefore(), 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, MakingVectors); pos != nil {
		t.Errorf("a pass without a listener is still being done: %+v", *pos)
	}
}
