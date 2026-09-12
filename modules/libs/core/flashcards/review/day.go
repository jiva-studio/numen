package review

import (
	"errors"
	"fmt"
	"time"
)

// DayStarts is how long past midnight a day of review begins by default.
//
// A person answering cards at one in the morning is finishing the day before,
// not starting the next one, and a boundary at midnight splits one session in
// two and calls half of it late.
const DayStarts = 4 * time.Hour

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
// falls at, because a person sits down when they sit down.
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

// EndOf is the instant the day holding at gives way to the next.
func (d Day) EndOf(at time.Time) time.Time {
	local := at.In(d.zone())
	y, m, day := local.Date()
	if opened := d.opens(y, m, day); local.Before(opened) {
		return opened
	}
	return d.opens(y, m, day+1)
}

// StartOf is the instant the day holding at began.
func (d Day) StartOf(at time.Time) time.Time {
	local := at.In(d.zone())
	y, m, day := local.Date()
	if opened := d.opens(y, m, day); !local.Before(opened) {
		return opened
	}
	return d.opens(y, m, day-1)
}

// Opened is the date the day holding at began on, as a plain date. A day is
// named and numbered from this, so that it is one day of review whatever the
// clock did around its boundaries.
func (d Day) Opened(at time.Time) time.Time {
	local := at.In(d.zone())
	y, m, day := local.Date()
	date := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	h, min := d.boundary()
	if local.Hour() < h || (local.Hour() == h && local.Minute() < min) {
		return date.AddDate(0, 0, -1)
	}
	return date
}

// opens is the instant the day of this date began. The boundary is an hour of
// the clock on the wall, so the day an hour was put into or taken out of begins
// and ends at the hour a person reads.
func (d Day) opens(y int, m time.Month, day int) time.Time {
	h, min := d.boundary()
	open := time.Date(y, m, day, h, min, 0, 0, d.zone())
	// An hour the clock skips over is read by no instant, and the day begins
	// where the clock jumped to.
	if late := time.Date(y, m, day, h, min, 0, 0, time.UTC).Sub(wall(open)); late > 0 {
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
func wall(at time.Time) time.Time {
	y, m, day := at.Date()
	return time.Date(y, m, day, at.Hour(), at.Minute(), 0, 0, time.UTC)
}

// Ending is the instant the day of this date gives way to the next. The date
// is read as it is written, and the boundary falls in the zone the days are
// counted in.
func (d Day) Ending(named time.Time) time.Time {
	y, m, day := named.Date()
	return d.opens(y, m, day+1)
}

// Owed reports whether a card is to be answered on this face in the day holding
// now. A card face that has never been answered is owed the first time it is
// asked about.
func (d Day) Owed(s Schedule, now time.Time) bool {
	return !s.Seen() || s.Due.Before(d.EndOf(now))
}
