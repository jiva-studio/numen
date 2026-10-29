package review

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// DayStartTime is how long past midnight a day of review begins by default.
//
// not starting the next one, and a boundary at midnight splits one session in
// two and calls half of it late.
const DayStartTime = 4 * time.Hour

// ErrNotAnHour is the error of an hour a day of review cannot be made to begin
// at.
var ErrNotAnHour = errors.New("a day of review begins at an hour of the day")

// Clock writes a length of time past midnight as an hour of the day.
func Clock(starts time.Duration) string {
	return fmt.Sprintf("%02d:%02d", int(starts.Hours()), int(starts.Minutes())%60)
}

// Day defines the boundaries of a day of review, where one day ends and the
// next begins.
//
// A card due on a day is due for the whole of it, whatever hour its schedule
// puts the due time at.
type Day struct {
	// Starts is how long past midnight a day begins.
	Starts time.Duration
	// In is the zone the day is counted in. A build holding none counts in the
	// machine's own.
	In *time.Location
}

// zone is where the days are counted. A day naming none counts in the
// machine's own.
func (d Day) zone() *time.Location {
	if d.In == nil {
		return time.Local
	}
	return d.In
}

// GetEnd is the instant the day holding at gives way to the next.
func (d Day) GetEnd(target time.Time) time.Time {
	local := target.In(d.zone())
	y, m, day := local.Date()
	if startTime := d.getStartTime(y, m, day); local.Before(startTime) {
		return startTime
	}
	return d.getStartTime(y, m, day+1)
}

// GetStart is the instant the day holding at began.
func (d Day) GetStart(target time.Time) time.Time {
	local := target.In(d.zone())
	y, m, day := local.Date()
	if startTime := d.getStartTime(y, m, day); !local.Before(startTime) {
		return startTime
	}
	return d.getStartTime(y, m, day-1)
}

// GetDate is the date of the day containing target, as a plain date. A day is
// named and numbered from this date, so it is one day of review regardless of
// clock changes at its boundaries.
func (d Day) GetDate(target time.Time) time.Time {
	local := target.In(d.zone())
	y, m, day := local.Date()
	date := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	h, minute := d.boundary()
	if local.Hour() < h || (local.Hour() == h && local.Minute() < minute) {
		return date.AddDate(0, 0, -1)
	}
	return date
}

// getStartTime is the instant the day of this date begins. The boundary is an
// hour of the wall clock in the zone the days are counted in, and the day ends
// at the same hour of the next date.
func (d Day) getStartTime(y int, m time.Month, day int) time.Time {
	h, minute := d.boundary()
	open := time.Date(y, m, day, h, minute, 0, 0, d.zone())
	// An hour the clock skips over is read by no instant, and the day begins
	// where the clock jumped to.
	if late := time.Date(y, m, day, h, minute, 0, 0, time.UTC).Sub(wall(open)); late > 0 {
		return open.Add(late)
	}
	return open
}

// boundary is the hour and minute of the clock a day begins at.
func (d Day) boundary() (int, int) {
	return int(d.Starts / time.Hour), int(d.Starts % time.Hour / time.Minute)
}

// wall is a clock reading with the zone removed, which allows two readings to be
// subtracted.
func wall(pos time.Time) time.Time {
	y, m, day := pos.Date()
	return time.Date(y, m, day, pos.Hour(), pos.Minute(), 0, 0, time.UTC)
}

// GetEndOfDate is the instant the day of this date gives way to the next. The
// date is read as it is written, and the boundary falls in the zone the days
// are counted in.
func (d Day) GetEndOfDate(named time.Time) time.Time {
	y, m, day := named.Date()
	return d.getStartTime(y, m, day+1)
}

// IsOwed reports whether a review card is due for answer in the day containing
// now. A review card that has never been answered is due the first time it is
// requested.
func (d Day) IsOwed(s Schedule, now time.Time) bool {
	return s.IsNew() || s.Due.Before(d.GetEnd(now))
}

// LatestDayStart is how far past midnight a day of review may be made to begin.
const LatestDayStart = 12 * time.Hour

// ParseDayStart parses the time past midnight at which a day of review begins
// from an `HH:MM` string. An empty string yields the default. A value that is
// not a time of day, or exceeds LatestDayStart, yields ErrNotAnHour and the
// default.
func ParseDayStart(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DayStartTime, nil
	}
	pos, err := time.Parse("15:04", raw)
	if err != nil {
		return DayStartTime, fmt.Errorf("%w, 00:00 to %s: %q", ErrNotAnHour, Clock(LatestDayStart), raw)
	}
	starts := time.Duration(pos.Hour())*time.Hour + time.Duration(pos.Minute())*time.Minute
	if starts > LatestDayStart {
		return DayStartTime, fmt.Errorf("%w, 00:00 to %s: %q", ErrNotAnHour, Clock(LatestDayStart), raw)
	}
	return starts, nil
}

// DayLayout is a day as it is written down and read back: the year, the month and
// the day it began on, in the zone the days are counted in.
const DayLayout = "2006-01-02"

// GetName is the name of the day of review that target falls in.
//
// It is the day the answer belongs to: an answer given at one in the morning
// belongs to the previous day when the review day begins after that hour, and
// is named for it.
func (d Day) GetName(target time.Time) string {
	return d.GetDate(target).Format(DayLayout)
}

// Tally is one day's answers: the number given, and the number for each of the
// four ratings.
//
// The ratings are kept apart because a day of fifty failed answers differs
// from a day of fifty successful ones, and both have the same total.
type Tally struct {
	Answered int
	Again    int
	Hard     int
	Good     int
	Easy     int
}

// GetDayTallies is what was answered on each day, by the name of the day.
//
// It is a sum and not an order: an answer arriving from another machine after
// later ones have been counted adds to the day it belongs to and disturbs
// nothing, which is what lets these be worked out one file at a time and kept.
func GetDayTallies(d Day, answers []Answer) map[string]Tally {
	undoTargets := make(map[string]bool)
	for _, a := range answers {
		if a.IsUndo() {
			undoTargets[a.Undoes] = true
		}
	}

	observed := make(map[string]bool, len(answers))
	out := make(map[string]Tally)
	for _, a := range answers {
		if a.IsUndo() || undoTargets[a.ID] || observed[a.ID] {
			continue
		}
		observed[a.ID] = true

		day := d.GetName(a.At)
		one := out[day]
		one.Answered++
		switch a.Rating {
		case Again:
			one.Again++
		case Hard:
			one.Hard++
		case Good:
			one.Good++
		case Easy:
			one.Easy++
		}
		out[day] = one
	}
	return out
}

// Spent is what a day came to under one preset: how many of its cards were
// answered, and how long those answers took.
//
// The counts are of cards and not of answers, which is the unit the budget is
// kept in: a card that comes round again in the same day is one card, and every
// answer given to it is time spent.
//
// New and Reviews divide the cards the way a budget does: a card is new on the
// day its review card is first answered at all, and a review on every day after.
type Spent struct {
	Answered int
	New      int
	Reviews  int
	Took     time.Duration
}

// GetSpentUnder is what the named day consumed under each preset, keyed by the
// path of the preset the review cards are grouped under.
//
// A review card with no preset is omitted, and so is an undone answer. Each
// answer is counted as Answer.Counted determines.
//
// counts gives the unit each preset counts in, keyed by the same path; a path
// not in counts is counted in cards: a review card counts once for the day
// however many times it is answered in it. Every answer contributes its time
// under either unit.
func GetSpentUnder(
	d Day, day string, answers []Answer,
	presetMap map[CardID]string, counts map[string]BudgetUnit,
) map[string]Spent {
	return OrderAnswers(answers).GetSpentUnder(d, day, presetMap, counts)
}

// GetSpentUnder is the same over a history already in order.
func (h History) GetSpentUnder(
	d Day, day string, presetMap map[CardID]string, counts map[string]BudgetUnit,
) map[string]Spent {
	// The review cards answered before the current answer, over the whole history
	// and not only this day.
	before := make(map[CardID]bool, len(h))
	// Which review cards the day has already counted, so a card that comes round
	// again in it is still the one card.
	counted := make(map[CardID]bool)
	out := make(map[string]Spent)
	for _, a := range h {
		first := !before[a.ReviewCard]
		before[a.ReviewCard] = true
		if d.GetName(a.At) != day {
			continue
		}
		path, target := presetMap[a.ReviewCard]
		if !target {
			continue
		}
		one := out[path]
		if counts[path].IsCharged(counted[a.ReviewCard]) {
			counted[a.ReviewCard] = true
			one.Answered++
			if first {
				one.New++
			} else {
				one.Reviews++
			}
		}
		if took := a.GetCountedTime(); took > 0 {
			one.Took += took
		}
		out[path] = one
	}
	return out
}

// GetReviewed is the review cards answered in the day named. An answer taken back is
// not one, and a review card answered again in the day is the one review card.
func GetReviewed(d Day, day string, answers []Answer) map[CardID]bool {
	return OrderAnswers(answers).GetReviewed(d, day)
}

// GetReviewed is the same over a history already in order.
func (h History) GetReviewed(d Day, day string) map[CardID]bool {
	out := make(map[CardID]bool)
	for _, a := range h {
		if d.GetName(a.At) == day {
			out[a.ReviewCard] = true
		}
	}
	return out
}

// Streak is the number of consecutive days up to now with at least one answer.
//
// A current day without answers does not end the streak: the streak is the
// run of days ending the previous day. A current day with answers is counted
// as part of the run.
func Streak(d Day, days map[string]Tally, now time.Time) int {
	pos := now.In(d.zone())
	if days[d.GetName(pos)].Answered == 0 {
		// Today is not answered yet, so the count is of the days behind it.
		pos = pos.AddDate(0, 0, -1)
	}

	out := 0
	for days[d.GetName(pos)].Answered > 0 {
		out++
		pos = pos.AddDate(0, 0, -1)
	}
	return out
}
