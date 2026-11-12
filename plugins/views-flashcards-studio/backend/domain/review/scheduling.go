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

func weekday(number int) time.Weekday {
	return time.Weekday(((number+int(time.Thursday))%7 + 7) % 7)
}

// AssignDueDay returns the day a card answered at this instant is due, and counts the card
// against that day. The day is chosen inside the tolerance around the scheduler's interval; each
// day weighs its day-of-week share of the load divided by the cards already due on it. The
// heaviest takes the card, the scheduler's day wins a tie, and a zero share takes none. The
// weight enforces no limit. A date preset moves no card. This is the only place a day is chosen.
func (p Preset) AssignDueDay(s *DueByDay, now, due time.Time) time.Time {
	out := p.computeDueDay(s, now, due)
	s.Add(out)
	return out
}

// GetDueDay returns the day a card answered at this instant would be due, without counting it
// against any day. The four answer buttons are four candidate answers for one card and one is
// applied: AssignDueDay chooses the day for the applied answer, and GetDueDay the day each
// button displays.
func (p Preset) GetDueDay(s *DueByDay, now, due time.Time) time.Time {
	return p.computeDueDay(s, now, due)
}

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

	// The instant is returned in the zone it arrived in.
	return due.In(in).AddDate(0, 0, on-dueDay).In(due.Location())
}

func (p Preset) getWeight(s *DueByDay, day int) float64 {
	return p.GetShare(weekday(day)) / float64(1+s.on[day])
}

// slack is the tolerance on either side of an interval within which a card may be moved, for the
// intervals from one length to another.
type slack struct{ from, to, factor float64 }

func getSlacks() []slack {
	return []slack{
		{2.5, 7, 0.15},
		{7, 20, 0.1},
		{20, math.Inf(1), 0.05},
	}
}

func window(away time.Duration) (first, last int, opens bool) {
	days := away.Hours() / 24
	if days < EvenFrom || days > EvenTo {
		return 0, 0, false
	}
	slack := 1.0
	for _, one := range getSlacks() {
		slack += one.factor * math.Max(math.Min(days, one.to)-one.from, 0)
	}
	first = int(math.Max(2, math.Round(days-slack)))
	last = int(math.Round(days + slack))
	return first, last, last > first
}
