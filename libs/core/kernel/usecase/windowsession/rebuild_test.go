package windowsession

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// announcing is an embedder that records the model it is read under and says when it changes.
type announcing struct {
	name, recipe string
	announce     func(isChunkSizeChanged bool)
}

func (a *announcing) Describe() (name, recipe string)                { return a.name, a.recipe }
func (a *announcing) OnModelChange(fn func(isChunkSizeChanged bool)) { a.announce = fn }

func TestTheRecordedModelFollowsTheModelTheEmbedderAnnounces(t *testing.T) {
	embedder := &announcing{name: "first", recipe: "mean-8"}
	w := newTestWindow(t, func(deps *Deps) { deps.Index.Model = embedder })
	w.open(t, garden())

	embedder.name, embedder.recipe = "second", "mean-16"
	embedder.announce(false)

	if name, recipe := w.rt.GetModel(); name != "second" || recipe != "mean-16" {
		t.Errorf("the window shows the model %q under %q, want second under mean-16", name, recipe)
	}
	if _, sources := w.wake.counts(); sources != 1 {
		t.Errorf("the sources were woken %d times, want once", sources)
	}
}

func TestAChangeOfTheChunkSizeBuildsThePassesOfTheVaultAgainOnce(t *testing.T) {
	embedder := &announcing{}
	w := newTestWindow(t, func(deps *Deps) { deps.Index.Model = embedder })
	w.open(t, garden())

	embedder.announce(true)

	if got := w.passes.vaults(); len(got) != 2 || got[1].ID != garden().ID {
		t.Errorf("the passes were started for %v, want the garden() twice", got)
	}
}

func TestAnAnnouncementWithTheSameChunkSizeBuildsNothingAgain(t *testing.T) {
	embedder := &announcing{}
	w := newTestWindow(t, func(deps *Deps) { deps.Index.Model = embedder })
	w.open(t, garden())

	embedder.announce(false)
	embedder.announce(false)

	if got := w.passes.vaults(); len(got) != 1 {
		t.Errorf("the passes were started %d times, want once", len(got))
	}
}

func TestAWindowWithoutAnEmbedderFollowsNothing(t *testing.T) {
	w := newTestWindow(t, nil)

	w.open(t, garden())

	if got := w.passes.vaults(); len(got) != 1 {
		t.Errorf("the passes were started %d times, want once", len(got))
	}
}

func TestAWindowThatShowsNoVaultBuildsNothingWhenTheChunkSizeChanges(t *testing.T) {
	embedder := &announcing{}
	w := newTestWindow(t, func(deps *Deps) { deps.Index.Model = embedder })
	w.open(t, domainNoVault())

	embedder.announce(true)

	if got := w.passes.vaults(); len(got) != 0 {
		t.Errorf("the passes were started for %v in a window with no vault", got)
	}
}

func domainNoVault() domain.Vault { return domain.Vault{} }
