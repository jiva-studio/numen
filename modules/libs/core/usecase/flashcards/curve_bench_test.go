package flashcards_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/review"
)

// curveLoad is the vault the window's requests are measured on, with its decks
// pointing at two presets: one deck points at the first and the other nineteen
// at the second.
func curveLoad(cards int) map[string]string {
	const settings = "goal: minutes_a_day\nminutes_a_day: 20\nnew_a_day: 8\n" +
		"reviews_a_day: 45\nretention: 0.87\n"
	out := loadDeck(cards)
	out["Small.md"] = "---\ntype: preset\n" + settings + "---\n"
	out["Large.md"] = "---\ntype: preset\n" + settings + "---\n"
	for at := range 20 {
		to := "Large"
		if at == 0 {
			to = "Small"
		}
		path := fmt.Sprintf("decks/Deck%02d.md", at)
		out[path] = strings.Replace(out[path], "---\ntype: deck\n",
			"---\ntype: deck\n"+names(to), 1)
	}
	return out
}

// curveWhole is the vault of twenty decks with every one of them pointing at
// one preset, so the curve of that preset is drawn over the whole of it.
func curveWhole(cards int) map[string]string {
	out := loadDeck(cards)
	out["Whole.md"] = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\n---\n"
	for at := range 20 {
		path := fmt.Sprintf("decks/Deck%02d.md", at)
		out[path] = strings.Replace(out[path], "---\ntype: deck\n",
			"---\ntype: deck\n"+names("Whole"), 1)
	}
	return out
}

// The curve of a preset holding the whole vault, at four sizes of it.
//
// What the projection costs stands on the card faces the preset holds, and
// these four rows are what says how it grows with them. The log grows with the
// vault too, at the same half an answer a card face the row above is built on.
//
// The schedule cache is filled before the clock starts, which is what opening
// the window and then a preset tab costs.
func BenchmarkCurveCards(b *testing.B) {
	for _, cards := range []int{500, 5000, 20000, 50000} {
		b.Run(fmt.Sprint(cards), func(b *testing.B) {
			s := opened(b, curveWhole(cards))
			loadAnswers(b, s, cards, 180, max(1, cards*150/50000))

			ctx := b.Context()
			if _, err := s.kept.Execute(ctx, s.vault); err != nil {
				b.Fatal(err)
			}
			p := review.Preset{
				Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			}
			curves := s.curves(time.Now())
			if _, err := curves.Execute(ctx, s.vault, "Whole.md", p); err != nil {
				b.Fatal(err)
			}
			for b.Loop() {
				if _, err := curves.Execute(ctx, s.vault, "Whole.md", p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The curve of one preset, over fifty thousand card faces in twenty decks
// answered a hundred and fifty times a day for six months.
//
// One preset holds a deck of the vault and the other holds the rest of it, so
// what the two rows are read against each other for is what a curve pays for
// the decks it does not schedule.
//
// The schedule cache is filled before the clock starts, which is what opening
// the window and then a preset tab costs.
func BenchmarkPresetCurve(b *testing.B) {
	const cards = 50000
	s := opened(b, curveLoad(cards))
	loadAnswers(b, s, cards, 180, 150)

	ctx := b.Context()
	if _, err := s.kept.Execute(ctx, s.vault); err != nil {
		b.Fatal(err)
	}
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	curves := s.curves(time.Now())

	for _, path := range []string{"Small.md", "Large.md"} {
		if _, err := curves.Execute(ctx, s.vault, path, p); err != nil {
			b.Fatal(err)
		}
		b.Run(strings.TrimSuffix(path, ".md"), func(b *testing.B) {
			for b.Loop() {
				if _, err := curves.Execute(ctx, s.vault, path, p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
