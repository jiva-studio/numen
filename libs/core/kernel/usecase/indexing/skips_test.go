package indexing

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

func getHeldWords() []string {
	return []string{"lighthouse", "beacon", "keeper", "lantern", "reef", "fog", "signal", "cliff"}
}

func getAfterWords() []string {
	return []string{"orchard", "pear", "ladder", "basket", "bloom", "frost", "graft", "hedge"}
}

// holdingModel blocks every batch with a text of the held note while isBlocking
// is set, and otherwise embeds it and counts the call.
type holdingModel struct {
	*fakeEmbedder
	started chan struct{}

	mu         sync.Mutex
	isBlocking bool
	heldCalls  int
}

func (m *holdingModel) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	isHeld := slices.ContainsFunc(texts, func(text string) bool { return strings.Contains(text, getHeldWords()[0]) })
	m.mu.Lock()
	isBlocking := m.isBlocking
	if isHeld && !isBlocking {
		m.heldCalls++
	}
	m.mu.Unlock()
	if isHeld && isBlocking {
		signal(m.started)
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return m.fakeEmbedder.Embed(ctx, texts)
}

func (*holdingModel) Concurrency() int { return 1 }

func (m *holdingModel) setBlocking(isBlocking bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isBlocking = isBlocking
}

func (m *holdingModel) getHeldCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heldCalls
}

func writeNote(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func testCanceledNoteIsSkippedUntilItChanges(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &holdingModel{fakeEmbedder: &fakeEmbedder{dims: 64}, started: make(chan struct{}, 1), isBlocking: true}
	made, db := openReading(t)
	fit(t, db, model)
	recipe := model.GetModel().Recipe()

	v := made.newVault(t, map[string]string{"Held.md": noteWith(getHeldWords(), 200)})
	cut(t, made, db, v)
	heldChunks, _, err := db.Progress.GetProgress(t.Context(), v.ID, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if heldChunks == 0 {
		t.Fatal("the held note was cut into no chunk that needs a vector")
	}

	p := newPass(made, db, events, model, recipe, v)
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		p.embed(t.Context())
	}()
	select {
	case <-model.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass never asked for the held note's vectors")
	}
	if !events.tasks.Cancel(work.MakingVectors) {
		t.Fatalf("the vector pass is not cancellable: %+v", events.tasks.List())
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass did not return after its task was canceled")
	}

	// The next wake embeds the other note and leaves the canceled one alone.
	model.setBlocking(false)
	writeNote(t, v.Path, "Other.md", noteWith(getBefore(), 200))
	cut(t, made, db, v)
	p.embed(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if calls := model.getHeldCalls(); calls != 0 {
		t.Errorf("the canceled note was sent to the model %d times", calls)
	}
	if target-embedded != heldChunks {
		t.Errorf("%d of %d chunks lack a vector, want the %d of the canceled note", target-embedded, target, heldChunks)
	}

	// An edit of the note, reported as the vault watch reports it, indexes it again.
	writeNote(t, v.Path, "Held.md", noteWith(slices.Concat(getAfterWords(), getHeldWords()[:1]), 300))
	p.worker.ClearSkips([]string{"Held.md"})
	cut(t, made, db, v)
	p.embed(t.Context())

	target, embedded, err = db.Progress.GetProgress(t.Context(), v.ID, recipe)
	if err != nil {
		t.Fatal(err)
	}
	if model.getHeldCalls() == 0 {
		t.Error("the edited note was not sent to the model")
	}
	if embedded != target {
		t.Errorf("after the edit %d of %d chunks carry a vector", embedded, target)
	}
}
