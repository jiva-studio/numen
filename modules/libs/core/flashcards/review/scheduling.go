package review

import (
	"math"
	"time"
)

// EvenFrom and EvenTo are the intervals a card may be moved within, in days.
// One falling short of the first or past the second stands where the scheduler
// put it.
const (
	EvenFrom = 2.5
	EvenTo   = 90
)

// DueByDay is how loaded each day of review is: how many card faces fall on
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

// Add counts one card face against the day its schedule falls in.
func (s *DueByDay) Add(due time.Time) {
	if s != nil {
		s.on[s.number(due)]++
	}
}

// CountOn is how many card faces fall on the day of review holding this instant.
func (s *DueByDay) CountOn(at time.Time) int {
	if s == nil {
		return 0
	}
	return s.on[s.number(at)]
}

// number is the day of review holding an instant, as a whole number counted
// from the day the clock is counted from. The same answers name the same days
// in every process.
func (s *DueByDay) number(at time.Time) int {
	return int(s.day.GetDate(at).Unix() / int64(24*time.Hour/time.Second))
}

// weekday is the day of the week a numbered day of review falls on. The day the
// clock is counted from was a Thursday.
func weekday(number int) time.Weekday {
	return time.Weekday(((number+int(time.Thursday))%7 + 7) % 7)
}

// Places is the day a card answered at this instant comes back on, and counts
// it against that day.
//
// It is chosen inside the tolerance the scheduler allows around the interval it
// worked out. Every day of that window carries a weight — the share of the load
// its day of the week keeps, over what already falls on it — and the heaviest
// takes the card, the day the scheduler named winning a tie. A day at nothing
// weighs nothing and takes no card.
//
// It is pressure and not a promise: no day is forbidden to carry more than its
// share, and a preset keeping no even load leaves the card where it fell.
//
// A preset aiming at a day moves no card face. The days beyond the front of a
// projection carry nothing, so the lightest day of a window is its last, and a
// card put there is a card asked for later.
//
// It is the one place a day is chosen. A session and a projection of it both
// come here.
func (p Preset) ScheduleDay(s *DueByDay, at, due time.Time) time.Time {
	out := p.computeLandingDay(s, at, due)
	s.Add(out)
	return out
}

// GetLandingDay is the day a card answered at this instant would come back on, counting
// it against no day.
//
// The four windows put to a person are four askings of one card, and one of
// them is answered. The day each of them names is chosen by Places' own
// arithmetic, so the button names the day the card lands on.
func (p Preset) GetLandingDay(s *DueByDay, at, due time.Time) time.Time {
	return p.computeLandingDay(s, at, due)
}

// computeLandingDay is where the day is chosen.
func (p Preset) computeLandingDay(s *DueByDay, at, due time.Time) time.Time {
	if s == nil {
		return due
	}
	first, last, opens := window(due.Sub(at))
	if !p.CanEvenLoad() || !opens {
		return due
	}

	// A day is added on the clock the boundary between days is read from. The
	// moment a card lands on is the same whichever zone the instant arrives in.
	in := s.day.zone()
	counted := at.In(in)

	stands := s.number(due)
	from, to := s.number(counted.AddDate(0, 0, first)), s.number(counted.AddDate(0, 0, last))
	on, heaviest := stands, -1.0
	if stands >= from && stands <= to {
		heaviest = p.getWeight(s, stands)
	}
	for day := from; day <= to; day++ {
		if weight := p.getWeight(s, day); weight > heaviest {
			on, heaviest = day, weight
		}
	}

	// The instant is handed back in the zone it arrived in.
	return due.In(in).AddDate(0, 0, on-stands).In(due.Location())
}

// getWeight is how much a numbered day of review wants another card: the share
// of the load its day of the week keeps, over what already falls on it.
func (p Preset) getWeight(s *DueByDay, day int) float64 {
	return p.GetShare(weekday(day)) / float64(1+s.on[day])
}

// slacks is how far either side of an interval a card may be put, by how long
// the interval is: the wider it is, the more days come round when a person
// expects them.
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
