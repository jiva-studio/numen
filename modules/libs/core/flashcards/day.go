package flashcards

import "time"

// DayStarts is how long past midnight a day of review begins by default.
//
// A person answering cards at one in the morning is finishing the day before,
// not starting the next one, and a boundary at midnight splits one sitting in
// two and calls half of it late.
const DayStarts = 4 * time.Hour

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

// Ends is the instant the day holding at gives way to the next.
func (d Day) Ends(at time.Time) time.Time {
	in := d.In
	if in == nil {
		in = time.Local
	}
	local := at.In(in)
	y, m, day := local.Date()
	// The boundary is an hour of the clock on the wall, so the day an hour was
	// put into or taken out of begins and ends at the hour a person reads.
	h, min := int(d.Starts/time.Hour), int(d.Starts%time.Hour/time.Minute)
	opened := time.Date(y, m, day, h, min, 0, 0, in)
	if local.Before(opened) {
		return opened
	}
	return time.Date(y, m, day+1, h, min, 0, 0, in)
}

// Owed reports whether a card is to be answered on this face in the day holding
// now. A card face that has never been answered is owed the first time it is
// asked about.
func (d Day) Owed(s Schedule, now time.Time) bool {
	return !s.Seen() || s.Due.Before(d.Ends(now))
}
