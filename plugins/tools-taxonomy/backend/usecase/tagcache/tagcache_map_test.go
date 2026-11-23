package tagcache_test

import (
	"maps"
	"slices"
	"testing"
)

func (s *scene) getMap(t *testing.T) map[string][]string {
	t.Helper()
	got, err := s.cache.GetMap(t.Context(), "garden")
	if err != nil {
		t.Fatal(err)
	}
	return got
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

	s.vault.WriteFile("a.md", "---\ntags: [lake]\n---\n")
	s.vault.WriteFile("b.md", "---\ntags: delta\n---\n")

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

func TestGetMapDropsNoteWithoutTags(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)

	s.vault.WriteFile("a.md", "# no tags\n")

	if got := s.getMap(t); len(got) != 0 {
		t.Errorf("map = %v, want no note", got)
	}
}

func TestGetMapDropsRemovedNote(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n", "b.md": "---\ntags: lake\n---\n"})
	s.getMap(t)

	s.vault.Remove("a.md")

	want := map[string][]string{"b.md": {"lake"}}
	if got := s.getMap(t); !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}

func TestGetMapUpdatesPathOfMovedNote(t *testing.T) {
	s := newScene(map[string]string{"a.md": "---\ntags: river\n---\n"})
	s.getMap(t)

	if _, err := s.vault.MoveNotes(t.Context(), []string{"a.md"}, "rivers"); err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{"rivers/a.md": {"river"}}
	if got := s.getMap(t); !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("map = %v, want %v", got, want)
	}
}
