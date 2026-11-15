package flashcards_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// A deleted preset note leaves the deck on the defaults, and the unresolved
// address is reported as a problem on the deck.
func TestADeckWhosePresetNoteIsMissingExistsOnTheDefaults(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckOf("Sanskrit", 20, 0),
	})
	if got := unseen(s.sessionAt(t, getToday(), getSaturday())); got != 2 {
		t.Fatalf("under the preset the day was asked %d new cards, want 2", got)
	}

	remove(t, s, "Sanskrit.md")

	target, err := s.getDeckPreset.Execute(t.Context(), s.vault, "decks/Roots.md")
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != "" {
		t.Errorf("read from %q", target.Path)
	}
	if !reflect.DeepEqual(target.Settings, review.Defaults()) {
		t.Errorf("preset = %+v", target.Settings)
	}
	if len(target.Problems) != 1 || !strings.Contains(target.Problems[0], "Sanskrit") {
		t.Errorf("problems = %v, want the one naming the note that is gone", target.Problems)
	}
	// The defaults are steered by their minutes, and twenty of them hold every
	// card the deck has left.
	if got := unseen(s.sessionAt(t, getToday(), getSaturday())); got != 20 {
		t.Errorf("on the defaults the day was asked %d new cards, want 20", got)
	}
}

// A deck naming two presets is scheduled by the first.
func TestSession_DeckWithTwoPresetsUsesFirstBudget(t *testing.T) {
	t.Parallel()
	s := openVault(t, map[string]string{
		"Term.md":        term,
		"Few.md":         preset("new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Many.md":        preset("new_a_day: 9\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Roots.md": deckNaming([]string{"Few", "Many"}, 20, 0),
	})

	if got := unseen(s.sessionAt(t, getToday(), getSaturday())); got != 2 {
		t.Errorf("the day was asked %d new cards, want the two the first preset keeps", got)
	}
}

// An index update that failed is not a write that failed. The file stat of the
// file the write produced is returned with it, and the caller's next save lands
