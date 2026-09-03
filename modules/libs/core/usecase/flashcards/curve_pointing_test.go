package flashcards_test

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// sanskrit is the identifier the preset of that name carries, so that a deck
// can name it the auxiliary way.
const sanskrit = "01M02ACGM0FYMSXNDP29C90JNR"

// pointed is a vault of three presets and five decks. Three decks name one
// preset — one of them by its identifier and two by its name — one deck names
// another preset, one names none, and the third preset is named by nothing.
func pointed() map[string]string {
	return map[string]string{
		"Term.md": vault["Term.md"],
		"Sanskrit.md": "---\ntype: preset\nid: " + sanskrit + "\ngoal: minutes_a_day\n" +
			"minutes_a_day: 20\nnew_a_day: 8\nreviews_a_day: 45\nretention: 0.87\n---\n\n# Sanskrit\n",
		"Grammar.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 15\n" +
			"new_a_day: 5\nreviews_a_day: 30\nretention: 0.9\n---\n\n# Grammar\n",
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 10\n" +
			"new_a_day: 4\nreviews_a_day: 20\nretention: 0.85\n---\n\n# Quiet\n",
		"decks/Roots.md":  pointingDeck(names("Sanskrit"), 0, 8),
		"decks/Verbs.md":  pointingDeck(names("Sanskrit"), 8, 8),
		"decks/Chants.md": pointingDeck(names("note://"+sanskrit), 16, 8),
		"decks/Cases.md":  pointingDeck(names("Grammar"), 24, 8),
		"decks/Other.md":  pointingDeck("", 32, 8),
	}
}

// names is the `links:` block of a deck naming its preset.
func names(to string) string {
	return "links:\n  - to: " + to + "\n    role: ref\n    type: preset\n"
}

// pointingDeck is a deck of cards cards, the first of them numbered from.
func pointingDeck(links string, from, cards int) string {
	out := "---\ntype: deck\n" + links + "---\n"
	for i := from; i < from+cards; i++ {
		out += fmt.Sprintf(
			"\n## Word %d ^%s\n\n[[Term]]\n\n### Word\n\nWord %d\n\n### Meaning\n\nMeaning %d\n",
			i, mark(i), i, i)
	}
	return out
}

// answers writes the same history into a vault: two cards of every deck
// answered three times each in the months before the day the curves are drawn
// on, with the times answers have — a middle of a few seconds and a tail of
// interruptions.
//
// Two vaults given this hold one history, so a curve over either is drawn from
// the same answers.
func answers(t *testing.T, s vaulted) {
	t.Helper()
	for _, days := range []int{150, 120, 90} {
		record := s.run(t, noon.AddDate(0, 0, -days))
		for card := 0; card < 40; card += 8 {
			answer(t, record, mark(card), 4*time.Second)
			answer(t, record, mark(card+1), 50*time.Second)
		}
	}
}

// The decks a curve is worked out over are the decks pointing at the preset,
// and how many of them there are is counted the same way.
func scheduling(t *testing.T, s vaulted, path string) (decks []string, faces int) {
	t.Helper()
	held, err := s.standings.Decks(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, deck := range held {
		p, err := s.presets.Of(t.Context(), s.vault, deck)
		if err != nil {
			t.Fatal(err)
		}
		if p.Path == path {
			decks = append(decks, deck)
		}
	}
	return decks, len(s.standings.Of(t.Context(), s.vault, decks))
}

// A curve reads the decks pointing at its preset and leaves the rest of the
// vault unread, and what it comes to is what reading the whole vault comes to:
// the same range, the same value at every place of it, the same marks, and the
// same decks and card faces behind them.
//
// The vault it is asked of holds decks of other presets and decks of none, and
// the one it is read against holds only the decks the preset schedules. Both
// hold one history, so what the two answers differ by is the reading.
func TestACurveIsDrawnFromTheDecksPointingAtThePreset(t *testing.T) {
	t.Parallel()
	shapes := []struct {
		name string
		path string
	}{
		{"a preset several decks point at", "Sanskrit.md"},
		{"a preset one deck points at", "Grammar.md"},
		{"a preset nothing points at", "Quiet.md"},
	}
	goals := []struct {
		name string
		p    history.Preset
	}{
		{"minutes", history.Preset{
			Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
		{"retention", history.Preset{
			Goal: history.GoalRetention, Retention: 0.87,
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
		{"a date", history.Preset{
			Goal: history.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Rule: history.RuleInterval, Interval: 21, Retention: 0.9,
		}},
	}

	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			whole := opened(t, pointed())
			answers(t, whole)
			decks, faces := scheduling(t, whole, shape.path)

			// The same vault with every deck the preset does not schedule taken
			// out of it, where there is nothing to narrow.
			notes := maps.Clone(pointed())
			for path := range notes {
				if strings.HasPrefix(path, "decks/") && !slices.Contains(decks, path) {
					delete(notes, path)
				}
			}
			alone := opened(t, notes)
			answers(t, alone)

			for _, goal := range goals {
				t.Run(goal.name, func(t *testing.T) {
					got, err := whole.curves(noon).
						Execute(t.Context(), whole.vault, shape.path, goal.p)
					if err != nil {
						t.Fatal(err)
					}
					want, err := alone.curves(noon).
						Execute(t.Context(), alone.vault, shape.path, goal.p)
					if err != nil {
						t.Fatal(err)
					}

					for _, one := range []flashcards.Curve{got, want} {
						if one.Decks != len(decks) || one.Cards != faces {
							t.Errorf("the curve carries %d decks and %d card faces, and %d decks "+
								"point at %s with %d card faces in them",
								one.Decks, one.Cards, len(decks), shape.path, faces)
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("over the whole vault:\n%s\nover its decks alone:\n%s",
							curveOf(got), curveOf(want))
					}
				})
			}
		})
	}
}

// curveOf is a curve in one line of a failure.
func curveOf(c flashcards.Curve) string {
	return fmt.Sprintf("%d decks, %d card faces, %d overdue, %d unbegun\ngrid %v\nnow %+v"+
		"\nsuggested %+v\nat %+v", c.Decks, c.Cards, c.Overdue, c.Unbegun, c.Grid,
		c.Now, c.Suggested, c.At)
}
