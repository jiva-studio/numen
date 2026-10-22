package flashcards_test

import (
	"testing"
	"time"
)

// The history screen opens each run file once. The days behind and the days
// ahead are two questions of one read.
func TestTheHistoryOpensEachRunFileOnce(t *testing.T) {
	t.Parallel()
	l := load(t, loadCards, loadDays, loadPerDay)
	ctx := t.Context()

	firstLoad := l.countLoads(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	if firstLoad.Opened != loadDays {
		t.Errorf("opened %d run files where the vault holds %d", firstLoad.Opened, loadDays)
	}
	if firstLoad.Listed != 1 {
		t.Errorf("listed the log %d times", firstLoad.Listed)
	}

	secondLoad := l.countLoads(t, func() error { _, err := l.review.Execute(ctx, l.vault); return err })
	if secondLoad.Opened != loadDays {
		t.Errorf("a warm request opened %d run files where the vault holds %d",
			secondLoad.Opened, loadDays)
	}
}

// What the screen shows is what it showed: the days, the streak, the answers,
// what is still to come and how much came back.
func TestTheHistoryShowsWhatItShowed(t *testing.T) {
	t.Parallel()
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

// The history counts one day of review for each day the vault was answered on,
// whatever the hour it is read at.
func TestTheHistoryCountsTheSameDaysAtAnyHour(t *testing.T) {
	t.Parallel()
	for _, at := range []time.Time{
		time.Date(2026, time.October, 17, 23, 59, 0, 0, time.UTC),
		time.Date(2026, time.October, 18, 0, 1, 0, 0, time.UTC),
		time.Date(2026, time.October, 18, 3, 45, 0, 0, time.UTC),
		time.Date(2026, time.October, 18, 4, 1, 0, 0, time.UTC),
	} {
		t.Run(at.Format("15:04"), func(t *testing.T) {
			t.Parallel()
			l := loadAt(t, func() time.Time { return at }, loadCards, loadDays, loadPerDay)

			got, err := l.review.Execute(t.Context(), l.vault)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Days) != loadDays {
				t.Errorf("counted %d days where the vault was answered on %d",
					len(got.Days), loadDays)
			}
			if got.Streak != loadDays {
				t.Errorf("a streak of %d where the vault was answered on the %d days before",
					got.Streak, loadDays)
			}
			if got.Answered != loadDays*loadPerDay {
				t.Errorf("counted %d answers where the vault holds %d",
					got.Answered, loadDays*loadPerDay)
			}
		})
	}
}
