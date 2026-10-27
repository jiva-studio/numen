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

// Day is where one day of review gives way to the next.
//
// A card owed today is owed for the whole of it, whatever hour its schedule
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

// GetDate is the date the day holding at began on, as a plain date. A day is
// named and numbered from this, so that it is one day of review whatever the
// clock did around its boundaries.
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

// getStartTime is the instant the day of this date began. The boundary is an hour of
// and ends at the hour a person reads.
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

// wall is a clock reading taken apart from the zone it was read in, so that two
// of them can be subtracted.
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

// IsOwed reports whether a card is to be answered on this review card in the day
// holding now. A review card that has never been answered is owed the first time
// it is asked about.
func (d Day) IsOwed(s Schedule, now time.Time) bool {
	return s.IsNew() || s.Due.Before(d.GetEnd(now))
}

// LatestDayStart is how far past midnight a day of review may be made to begin.
const LatestDayStart = 12 * time.Hour

// ParseDayStart is how long past midnight a day of review begins, from the hour a person wrote it as,
// `HH:MM`. Nothing written is the default. Anything that is not an hour of the day, or is past
// LatestDayStart, is ErrNotAnHour, and the default stands with it.
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

// GetName is the name of the day an answer given at this instant falls in.
//
// It is the day a person would say they answered on, which is the day the
// review day began — an answer given at one in the morning belongs to the day
// before, and is named for it.
func (d Day) GetName(target time.Time) string {
	return d.GetDate(target).Format(DayLayout)
}

// Tally is one day's answers: how many were given, and how each of the four was
// said.
//
// The four are kept apart because a day of fifty cards a person could not
// recall is a different day from fifty they could, and the two are the same
// number.
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

// GetSpentUnder is what the day named came to under each preset, by the path
// the review cards are grouped under.
//
// A review card nothing groups is left out, and so is an answer taken back. Each
// answer counts for what Answer.Counted makes of it.
//
// Counts says how each preset counts, by the same path, and a path it does not
// name counts in cards: a review card counts once for the day however many times
// it is answered in it. Every one of those answers counts its time either way.
func GetSpentUnder(
	d Day, day string, answers []Answer,
	presetMap map[ReviewCardID]string, counts map[string]BudgetUnit,
) map[string]Spent {
	return OrderAnswers(answers).GetSpentUnder(d, day, presetMap, counts)
}

// GetSpentUnder is the same over a history already in order.
func (h History) GetSpentUnder(
	d Day, day string, presetMap map[ReviewCardID]string, counts map[string]BudgetUnit,
) map[string]Spent {
	// Which review cards have been answered before the answer in hand, over the
	// whole history and not this day alone.
	before := make(map[ReviewCardID]bool, len(h))
	// Which review cards the day has already counted, so a card that comes round
	// again in it is still the one card.
	counted := make(map[ReviewCardID]bool)
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
func GetReviewed(d Day, day string, answers []Answer) map[ReviewCardID]bool {
	return OrderAnswers(answers).GetReviewed(d, day)
}

// GetReviewed is the same over a history already in order.
func (h History) GetReviewed(d Day, day string) map[ReviewCardID]bool {
	out := make(map[ReviewCardID]bool)
	for _, a := range h {
		if d.GetName(a.At) == day {
			out[a.ReviewCard] = true
		}
	}
	return out
}

// Streak is how many days up to now were reviewed without a gap.
//
// on yet does not end a streak: a person who has not sat down this morning has
// not broken anything, and their streak is what they had last night. A day they
// did answer on counts from itself.
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
