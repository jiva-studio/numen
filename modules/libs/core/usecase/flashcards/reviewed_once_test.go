package flashcards_test

import (
	"testing"
)

// The history screen opens each run file once. The days behind and the days
// ahead are two questions of one reading.
func TestTheHistoryOpensEachRunFileOnce(t *testing.T) {
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	cold := l.counting(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	if cold.Opened != loadDays {
		t.Errorf("opened %d run files where the vault holds %d", cold.Opened, loadDays)
	}
	if cold.Listed != 1 {
		t.Errorf("listed the log %d times", cold.Listed)
	}

	warm := l.counting(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	if warm.Opened != loadDays {
		t.Errorf("a warm request opened %d run files where the vault holds %d",
			warm.Opened, loadDays)
	}
}

// What the screen shows is what it showed: the days, the streak, the answers,
// what is still to come and how much came back.
func TestTheHistoryShowsWhatItShowed(t *testing.T) {
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	first, err := l.review.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}
	again, err := l.review.Execute(ctx, l.vault)
	if err != nil {
		t.Fatal(err)
	}

	if first.Answered != loadDays*loadPerDay {
		t.Errorf("counted %d answers where the vault holds %d",
			first.Answered, loadDays*loadPerDay)
	}
	if len(first.Days) != loadDays {
		t.Errorf("counted %d days where the vault was answered on %d",
			len(first.Days), loadDays)
	}
	if first.Answered != again.Answered || first.Streak != again.Streak {
		t.Errorf("%d answers over a streak of %d, then %d over %d",
			first.Answered, first.Streak, again.Answered, again.Streak)
	}
	if len(first.Due) != len(again.Due) || len(first.Retained) != len(again.Retained) {
		t.Errorf("%d days ahead and %d behind, then %d and %d",
			len(first.Due), len(first.Retained), len(again.Due), len(again.Retained))
	}
	for day, was := range first.Days {
		if now := again.Days[day]; now != was {
			t.Errorf("%s came to %+v, and was %+v", day, now, was)
		}
	}
	for day, was := range first.Retained {
		if now := again.Retained[day]; now != was {
			t.Errorf("%s kept %+v, and was %+v", day, now, was)
		}
	}
}
