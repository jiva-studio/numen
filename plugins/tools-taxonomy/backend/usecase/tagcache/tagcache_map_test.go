package tagcache_test

import (
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/usecase/tagcache"
)

const waitForChange = 2 * time.Second

func (s *scene) getMap(t *testing.T) map[string][]string {
	t.Helper()
	got, err := s.cache.GetMap(t.Context(), "garden")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func (s *scene) subscribe(t *testing.T) (<-chan tagcache.Change, func()) {
	t.Helper()
	_, changes, cancel, err := s.cache.Watch(t.Context(), "garden")
	if err != nil {
		t.Fatal(err)
	}
	return changes, cancel
}

func receive(t *testing.T, changes <-chan tagcache.Change) (tagcache.Change, bool) {
	t.Helper()
	select {
	case change, ok := <-changes:
		return change, ok
	case <-time.After(waitForChange):
		t.Error("no change arrived")
		return tagcache.Change{}, false
	}
}

func TestGetMapReturnsTheNotesWithTags(t *testing.T) {
	s := newScene(map[string]string{
		"a.md": "---\ntags: [river, delta]\n---\n", "b.md": "---\ntags: river\n---\n", "c.md": "# no tags\n",
	})

	got := s.getMap(t)

	want := map[string][]string{"a.md": {"river", "delta"}, "b.md": {"river"}}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}

func TestTheMapIsBroughtUpToDateBeforeItIsGiven(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)

	s.vault.File("a.md", "---\ntags: [lake]\n---\n")
	s.vault.File("b.md", "---\ntags: delta\n---\n")

	want := map[string][]string{"a.md": {"lake"}, "b.md": {"delta"}}
	if got := s.getMap(t); !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}

func TestTheMapReadsTheNotesOnceForAnyNumberOfAnswers(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getMap(t)
	s.reads.take()

	s.getMap(t)
	s.getMap(t)

	if asked := s.reads.take(); len(asked) != 0 {
		t.Errorf("answering the map again read %+v", asked)
	}
}

func TestAChangedNoteIsSentToTheSubscriberWithItsTags(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()

	s.vault.File("a.md", "---\ntags: [river, delta]\n---\n")
	s.update(t)

	change, ok := receive(t, changes)
	want := map[string][]string{"a.md": {"river", "delta"}}
	if !ok || !maps.EqualFunc(change.Changed, want, slices.Equal) {
		t.Errorf("change = %v, want %v", change.Changed, want)
	}
}

func TestANoteThatLostItsTagsIsSentWithAnEmptyList(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()

	s.vault.File("a.md", "# no tags\n")
	s.update(t)

	change, ok := receive(t, changes)
	if tags, has := change.Changed["a.md"]; !ok || !has || len(tags) != 0 || len(change.Changed) != 1 {
		t.Errorf("change = %v, want a.md with an empty list", change.Changed)
	}
	if got := s.getMap(t); len(got) != 0 {
		t.Errorf("map = %v, want no note", got)
	}
}

func TestARemovedNoteIsSentWithAnEmptyListAndDroppedFromTheMap(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()

	s.vault.Remove("a.md")
	s.update(t)

	change, ok := receive(t, changes)
	if tags, has := change.Changed["a.md"]; !ok || !has || len(tags) != 0 || len(change.Changed) != 1 {
		t.Errorf("change = %v, want a.md with an empty list", change.Changed)
	}
	want := map[string][]string{"b.md": {"lake"}}
	if got := s.getMap(t); !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}

func TestAMovedNoteIsSentAtItsNewPathAndEmptyAtTheOldPath(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()

	if _, err := s.vault.MoveNotes(t.Context(), []string{"a.md"}, "rivers"); err != nil {
		t.Fatal(err)
	}
	s.update(t)

	change, ok := receive(t, changes)
	old, hasOld := change.Changed["a.md"]
	if !ok || !hasOld || len(old) != 0 || !slices.Equal(change.Changed["rivers/a.md"], []string{"river"}) || len(change.Changed) != 2 {
		t.Errorf("change = %v, want a.md empty and rivers/a.md with river", change.Changed)
	}
	want := map[string][]string{"rivers/a.md": {"river"}}
	if got := s.getMap(t); !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}

func TestAnUpdateThatChangesNoTagsSendsNothing(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()

	s.vault.File("a.md", "---\ntags: river\n---\nA new body.\n")
	s.update(t)

	select {
	case change := <-changes:
		t.Errorf("an update that left the tags as they were sent %v", change.Changed)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestEverySubscriberGetsTheChange(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	first, cancelFirst := s.subscribe(t)
	defer cancelFirst()
	second, cancelSecond := s.subscribe(t)
	defer cancelSecond()

	s.vault.File("a.md", "---\ntags: lake\n---\n")
	s.update(t)

	want := map[string][]string{"a.md": {"lake"}}
	for _, changes := range []<-chan tagcache.Change{first, second} {
		if change, ok := receive(t, changes); !ok || !maps.EqualFunc(change.Changed, want, slices.Equal) {
			t.Errorf("change = %v, want %v", change.Changed, want)
		}
	}
}

func TestACanceledSubscriberIsClosedAndDoesNotHoldUpTheUpdates(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	if changes == nil {
		t.Fatal("Watch gave no channel")
	}

	cancel()
	s.vault.File("a.md", "---\ntags: lake\n---\n")
	s.update(t)

	select {
	case change, ok := <-changes:
		if ok {
			t.Errorf("a canceled subscriber got %v", change.Changed)
		}
	case <-time.After(waitForChange):
		t.Error("the channel of a canceled subscriber was not closed")
	}
}

func TestASubscriberThatReadsNothingDoesNotHoldUpTheUpdates(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	changes, cancel := s.subscribe(t)
	defer cancel()
	if changes == nil {
		t.Fatal("Watch gave no channel")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := range 50 {
			s.vault.File("a.md", "---\ntags: ["+string(rune('a'+i%26))+"x]\n---\n")
			if err := s.cache.UpdateVault(t.Context(), "garden"); err != nil {
				t.Error(err)
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("updates stopped while a subscriber read nothing")
	}
	if got := s.getMap(t); len(got["a.md"]) != 1 {
		t.Errorf("map = %v, want the last tags of a.md", got)
	}
}

func TestAWatchSendsNoChangeThatIsInTheMapItStartedWith(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)
	s.vault.File("a.md", "---\ntags: [lake]\n---\n")

	got, changes, cancel, err := s.cache.Watch(t.Context(), "garden")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	s.vault.File("b.md", "---\ntags: delta\n---\n")
	s.update(t)

	want := map[string][]string{"a.md": {"lake"}}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
	change, ok := receive(t, changes)
	wantChange := map[string][]string{"b.md": {"delta"}}
	if !ok || !maps.EqualFunc(change.Changed, wantChange, slices.Equal) {
		t.Errorf("change = %v, want %v", change.Changed, wantChange)
	}
}
