package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// Review is what a day of review is, on this person's clock.
type Review struct {
	// DayStarts is the hour a day of review begins at, on the clock on the
	// wall, written as hours and minutes. An answer given before it is written
	// into the day before.
	DayStarts string `json:"day_starts"`
}

// LatestDayStarts is how far past midnight a day may be made to begin.
const LatestDayStarts = 12 * time.Hour

// ClockFormat is how an hour of the day is written.
const ClockFormat = "15:04"

// DayStart returns how long past midnight a day of review begins, and whether the file
// said something that is not an hour of the day.
func (r Review) DayStart() (time.Duration, bool) {
	written := strings.TrimSpace(r.DayStarts)
	if written == "" {
		return DefaultDayStart(), true
	}
	at, err := time.Parse(ClockFormat, written)
	if err != nil {
		return DefaultDayStart(), false
	}
	starts := time.Duration(at.Hour())*time.Hour + time.Duration(at.Minute())*time.Minute
	if starts > LatestDayStarts {
		return DefaultDayStart(), false
	}
	return starts, true
}

// DefaultDayStart returns when a day of review begins where the file says nothing. An
// answer given before it finishes the evening it belongs to.
func DefaultDayStart() time.Duration { return review.DayStarts }

// ReadDayStart is the hour a day of review is to begin at, as it goes into the
// file. An hour past LatestDayStarts, anything that is not an hour of the
// clock, and no hour at all, are review.ErrNotAnHour. It reads and writes
// no file.
func ReadDayStart(written string) (string, error) {
	starts, hour := Review{DayStarts: written}.DayStart()
	if !hour || strings.TrimSpace(written) == "" {
		return "", fmt.Errorf("%w, 00:00 to %s: %q",
			review.ErrNotAnHour, review.Clock(LatestDayStarts), written)
	}
	return review.Clock(starts), nil
}
