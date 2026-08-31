package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// The requests a window makes of a vault, over fifty thousand card faces
// answered a hundred and fifty times a day for six months.
//
// The first three are warm: the caches are filled before the clock starts,
// which is what a person's second question of an evening costs. The last is the
// history screen on a build that keeps no counting, which is what the first
// question of a launch costs.
func BenchmarkVault(b *testing.B) {
	l := load(b, 50000, 180, 150)
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

	uncounted := flashcards.Counted{
		Logs: l.review.Logs, Schedules: l.review.Schedules, Day: today, Now: time.Now,
	}
	b.Run("HistoryUncounted", func(b *testing.B) {
		for b.Loop() {
			if _, err := uncounted.Execute(ctx, l.vault); err != nil {
				b.Fatal(err)
			}
		}
	})
}
