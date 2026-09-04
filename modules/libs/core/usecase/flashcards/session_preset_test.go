package flashcards_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// over is what the vault asks at this instant when the sitting is opened over
// this deck or this preset.
func (s vaulted) over(
	t *testing.T, day review.Day, now time.Time, at flashcards.Scope,
) (flashcards.Sitting, error) {
	t.Helper()
	return flashcards.Session{
		Marking: s.marking, Standings: s.standings, Schedules: s.kept,
		Presets: s.presets, Day: day, Now: func() time.Time { return now },
	}.Execute(t.Context(), s.vault, at)
}

// under is the sitting over one preset, and a fatal error where it was refused.
func (s vaulted) under(
	t *testing.T, day review.Day, now time.Time, preset string,
) flashcards.Sitting {
	t.Helper()
	sat, err := s.over(t, day, now, flashcards.ByPreset(preset))
	if err != nil {
		t.Fatal(err)
	}
	return sat
}

// Pressing a preset sits to the cards of every deck pointing at it, and to no
// card of another preset.
func TestASittingOverAPresetAsksTheDecksThatPointAtIt(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":         term,
		"Steady.md":       preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Other.md":        preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md":  deckOf("Steady", 3, 0),
		"decks/Trees.md":  deckOf("Steady", 4, 100),
		"decks/Rivers.md": deckOf("Other", 5, 200),
	})

	got := byDeck(s.under(t, today, saturday, "Steady.md"))
	want := map[string]int{"decks/Birds.md": 3, "decks/Trees.md": 4}
	for deck, cards := range want {
		if got[deck] != cards {
			t.Errorf("%s was asked %d cards, want %d", deck, got[deck], cards)
		}
	}
	if len(got) != len(want) {
		t.Errorf("the sitting held %v, want %v", got, want)
	}
}

// The decks naming no preset are a preset of their own, and the sitting over it
// is opened by naming no note.
func TestASittingOverThePresetOfTheDecksNamingNone(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 20\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Loose.md": deckOf("", 3, 0),
		"decks/Birds.md": deckOf("Steady", 4, 100),
	})

	got := byDeck(s.under(t, today, saturday, ""))
	if len(got) != 1 || got["decks/Loose.md"] != 3 {
		t.Errorf("the sitting held %v, want three cards of decks/Loose.md", got)
	}
}

// The cards a preset offers are what its own allowance admits for the day, and
// the decks under it share that one budget.
func TestASittingOverAPresetIsHeldToItsBudget(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Five.md":        preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Five", 10, 0),
		"decks/Trees.md": deckOf("Five", 10, 100),
	})

	if got := asked(s.under(t, today, saturday, "Five.md")); got != 5 {
		t.Errorf("a preset of five new cards a day offered %d", got)
	}
}

// A second sitting over the same preset takes up where the first left off: what
// the day has spent is off the allowance the tile was drawn from.
func TestASecondSittingOverAPresetTakesUpWhereTheFirstLeftOff(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":       term,
		"Five.md":       preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Five.md": deckOf("Five", 20, 0),
	})
	if got := unseen(s.under(t, today, saturday, "Five.md")); got != 5 {
		t.Fatalf("the morning was asked %d new cards, want 5", got)
	}

	morning := s.run(t, saturday)
	for i := range 3 {
		answer(t, morning, mark(i), 6*time.Second)
	}

	if got := unseen(s.under(t, today, saturday.Add(2*time.Hour), "Five.md")); got != 2 {
		t.Errorf("the evening was asked %d new cards, want 2", got)
	}
}

// A deck and a preset are two answers to which cards were meant, and the
// sitting puts the question back.
func TestNamingADeckAndAPresetTogetherIsRefused(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Steady", 3, 0),
	})

	_, err := s.over(t, today, saturday, flashcards.Scope{
		Deck: "decks/Birds.md", Preset: "Steady.md", Named: true,
	})
	if !errors.Is(err, flashcards.ErrBothNamed) {
		t.Fatalf("a deck and a preset together were answered with %v", err)
	}
}

// A preset no deck points at has nothing to sit to, and says so.
func TestASittingOverAPresetNothingPointsAtIsRefused(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":        term,
		"Steady.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"Lonely.md":      preset("new_a_day: 5\nreviews_a_day: 0\nminutes_a_day: 0\n"),
		"decks/Birds.md": deckOf("Steady", 3, 0),
	})

	_, err := s.over(t, today, saturday, flashcards.ByPreset("Lonely.md"))
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a preset nothing points at was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "no deck") {
		t.Errorf("the reason shown is %q", err)
	}
}

// A paused preset is refused with the reason, and not with an empty sitting.
func TestASittingOverAPausedPresetIsRefused(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md":         term,
		"Paused.md":       preset("new_a_day: 0\nreviews_a_day: 0\n"),
		"decks/Paused.md": deckOf("Paused", 6, 0),
	})

	_, err := s.over(t, today, saturday, flashcards.ByPreset("Paused.md"))
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a paused preset was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "paused") {
		t.Errorf("the reason shown is %q", err)
	}
}

// A day already spent is refused with the reason. The first sitting takes the
// whole of the allowance, and the second is told why there is nothing left.
func TestASittingOverAPresetWhoseDayIsSpentIsRefused(t *testing.T) {
	t.Parallel()
	s := opened(t, map[string]string{
		"Term.md": term,
		"Two.md": preset(
			"new_a_day: 2\nreviews_a_day: 0\nminutes_a_day: 0\ncounts: shows\n"),
		"decks/Two.md":  deckOf("Two", 6, 0),
		"decks/Rest.md": deckOf("", 6, 100),
	})
	if got := unseen(s.under(t, today, saturday, "Two.md")); got != 2 {
		t.Fatalf("the morning was asked %d new cards, want 2", got)
	}

	morning := s.run(t, saturday)
	for i := range 2 {
		answer(t, morning, mark(i), 6*time.Second)
	}

	_, err := s.over(t, today, saturday.Add(2*time.Hour), flashcards.ByPreset("Two.md"))
	if !errors.Is(err, flashcards.ErrSchedulesNothing) {
		t.Fatalf("a spent day was answered with %v", err)
	}
	if !strings.Contains(err.Error(), "spent") {
		t.Errorf("the reason shown is %q", err)
	}
}
