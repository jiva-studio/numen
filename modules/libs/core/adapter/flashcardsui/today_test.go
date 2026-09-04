package flashcardsui

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/review"
)

// The counts the window is handed stand in a review day, and the window weighs
// a preset's goal against that day. A day of review begins at the hour the
// settings name, so between midnight and that hour the calendar has turned and
// the day has not.
func TestTheDayTheCountsStandInIsTheReviewDay(t *testing.T) {
	api, _ := windowed(t, deck)
	api.Day = review.Day{Starts: 4 * time.Hour, In: time.UTC}

	for name, c := range map[string]struct {
		at   time.Time
		want string
	}{
		"an hour past midnight": {
			time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC), "2026-09-04",
		},
		"the hour the day begins at": {
			time.Date(2026, 9, 5, 4, 0, 0, 0, time.UTC), "2026-09-05",
		},
		"the middle of the day": {
			time.Date(2026, 9, 5, 14, 0, 0, 0, time.UTC), "2026-09-05",
		},
	} {
		t.Run(name, func(t *testing.T) {
			api.Now = func() time.Time { return c.at }
			if got := front(t, api).GetDay(); got != c.want {
				t.Errorf("at %v the counts stand in %q, want %q", c.at, got, c.want)
			}
		})
	}
}
