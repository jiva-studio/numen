package review

import (
	"math"
	"time"
)

// EvenFrom and EvenTo bound the intervals, in days, within which a card may be
// moved. An interval below EvenFrom or above EvenTo keeps the due day the
// scheduler chose.
const (
	EvenFrom = 2.5
	EvenTo   = 90
)

// DueByDay is how loaded each day of review is: how many review cards fall on
// each.
//
// It is one table over every preset, and the answers replayed and the
// projection ahead of them both read it.
type DueByDay struct {
	day Day
	on  map[int]int
}

// NewDueByDay opens a table counting the days as this day of review divides
// them.
func NewDueByDay(d Day) *DueByDay { return &DueByDay{day: d, on: make(map[int]int)} }

// Add counts one review card against the day its schedule falls in.
func (s *DueByDay) Add(due time.Time) {
	if s != nil {
		s.on[s.GetDayNumber(due)]++
	}
}

// CountOn is how many review cards fall on the day of review holding this instant.
func (s *DueByDay) CountOn(target time.Time) int {
	if s == nil {
		return 0
	}
	return s.on[s.GetDayNumber(target)]
}

// GetDayNumber is the day of review holding an instant, as a whole number counted
// from the day the clock is counted from. The same answers name the same days
// in every process.
func (s *DueByDay) GetDayNumber(pos time.Time) int {
	return int(s.day.GetDate(pos).Unix() / int64(24*time.Hour/time.Second))
}

// weekday is the day of the week a numbered day of review falls on. The day the
// clock is counted from was a Thursday.
func weekday(number int) time.Weekday {
	return time.Weekday(((number+int(time.Thursday))%7 + 7) % 7)
}

// AssignDueDay returns the day a card answered at this instant is due, and
// counts the card against that day.
//
// The day is chosen inside the tolerance the scheduler allows around the
// interval it computed. Every day of that window has a weight, the share of
// the load its day of the week carries divided by the number of cards already
// due on it, and the heaviest day takes the card; the day the scheduler chose
// wins a tie. A day with zero share has zero weight and takes no card.
//
// The weight biases the choice and does not enforce a limit: no day is
// prevented from carrying more than its share, and a preset with an uneven
// load disabled keeps the card on the day the scheduler chose.
//
// A preset aiming at a date moves no review card. The days beyond the front of
// a projection carry nothing, so the lightest day of a window is its last, and
// a card moved there is presented later.
//
// This is the only place a day is chosen. A session and its projection both
// call it.
func (p Preset) AssignDueDay(s *DueByDay, now, due time.Time) time.Time {
	out := p.computeDueDay(s, now, due)
	s.Add(out)
	return out
}

// GetDueDay returns the day a card answered at this instant would be due,
// without counting it against any day.
//
// The four answer options shown to the user are four candidate answers for one
// card, and one of them is applied. AssignDueDay chooses the day for the
// applied answer, and GetDueDay chooses the day each button displays.
func (p Preset) GetDueDay(s *DueByDay, now, due time.Time) time.Time {
	return p.computeDueDay(s, now, due)
}

// computeDueDay is where the day is chosen.
func (p Preset) computeDueDay(s *DueByDay, pos, due time.Time) time.Time {
	if s == nil {
		return due
	}
	first, last, opens := window(due.Sub(pos))
	if !p.ShouldBalanceLoad() || !opens {
		return due
	}

	// A day is added on the clock the boundary between days is read from. The
	// moment a card lands on is the same whichever zone the instant arrives in.
	in := s.day.zone()
	counted := pos.In(in)

	dueDay := s.GetDayNumber(due)
	from, to := s.GetDayNumber(counted.AddDate(0, 0, first)), s.GetDayNumber(counted.AddDate(0, 0, last))
	on, heaviest := dueDay, -1.0
	if dueDay >= from && dueDay <= to {
		heaviest = p.getWeight(s, dueDay)
	}
	for day := from; day <= to; day++ {
		if weight := p.getWeight(s, day); weight > heaviest {
			on, heaviest = day, weight
		}
	}

	// The instant is handed back in the zone it arrived in.
	return due.In(in).AddDate(0, 0, on-dueDay).In(due.Location())
}

// getWeight is how much a numbered day of review needs another card: the share
// of the load its day of the week carries, divided by the cards already due on
// it plus one.
func (p Preset) getWeight(s *DueByDay, day int) float64 {
	return p.GetShare(weekday(day)) / float64(1+s.on[day])
}

// slacks is the tolerance on either side of an interval within which a card may
// be moved, by interval length: the longer the interval, the larger the
// tolerance in days.
var slacks = []struct{ from, to, factor float64 }{
	{2.5, 7, 0.15},
	{7, 20, 0.1},
	{20, math.Inf(1), 0.05},
}

// window is the days either side of an interval a card may be put on, counted
// from the answer, and whether the window holds more than one day.
func window(away time.Duration) (first, last int, opens bool) {
	days := away.Hours() / 24
	if days < EvenFrom || days > EvenTo {
		return 0, 0, false
	}
	slack := 1.0
	for _, one := range slacks {
		slack += one.factor * math.Max(math.Min(days, one.to)-one.from, 0)
	}
	first = int(math.Max(2, math.Round(days-slack)))
	last = int(math.Round(days + slack))
	return first, last, last > first
}
