package flashcards_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// The three requests a window makes of a vault, over twenty thousand card faces
// answered a hundred and fifty times a day for six months.
//
// Every one of them is warm: the caches are filled before the clock starts,
// which is what a person's second question of an evening costs.
func BenchmarkVault(b *testing.B) {
	l := load(b, 20000, 180, 150)
	ctx := b.Context()
	if _, err := l.owed.Execute(ctx, l.vault); err != nil {
		b.Fatal(err)
	}
	if _, err := l.review.Execute(ctx, l.vault); err != nil {
		b.Fatal(err)
	}

	b.Run("FrontDoor", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.owed.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Sitting", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.sat.Execute(ctx, l.vault, flashcards.Over{}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("History", func(b *testing.B) {
		for b.Loop() {
			if _, err := l.review.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})
}
