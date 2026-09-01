package flashcards_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
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
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
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
