package review

import (
	"math"
	"time"
)

// Ahead is how many days a projection runs when it is not told.
const Ahead = 90

// LeastStability is the least a projected card face is taken to stand at, in
// days, which is a minute. The scheduler divides by stability, so a card face a
// long run of lapses has worn down to none of it answers with a number nobody
// can read, and every figure weighed against it after that is that number.
const LeastStability = 1.0 / (24 * 60)

// MostShowings is how many times one day of review asks a card face. An answer
// a card did not come back on sends it away for minutes, and the day it lands
// back in is the day it was asked in; a card that keeps landing there is put
// down and picked up by the day after.
const MostShowings = 8

// Simulation projects a preset forward over the days ahead: its card faces
// answered day after day, inside the budgets it keeps.
//
// The share of the load each day of the week carries scales what that day
// admits, and an even load moves cards onto the days carrying least. Both are
// Preset's, and the session reads them the same way.
type Simulation struct {
	By   Scheduler
	Day  Day
	Cost AnswerCost
	// Days is how far ahead it runs, and runs Ahead days when it is zero.
	Days int
	// Retains is the days of the run whose returning share it works out, counting
	// the day it opens as none. A day nobody names is not worked out, and the
	// projection answers for none of it.
	//
	// The share is a pass over every card face the preset holds, and it is the
	// one thing a day counts over the whole material.
	Retains []int
	// Recalls is what this run assumes about coming back. A run holding none
	// reads AsModelled.
	Recalls RecallChance
	// Spent is what the day holding now has already gone through under this
	// preset. The first day of a run is a real day a person may be halfway
	// through, and what it has left is what a session opened now would offer.
	Spent Spent
}

// Covers is how many days this run walks, counting the day it opens as the
// first. A run told nothing walks Ahead of them.
func (s Simulation) Covers() int {
	if s.Days <= 0 {
		return Ahead
	}
	return s.Days
}

// NeverRipens is a rule a card face begun now does not reach in the years
// Ripens looks over.
const NeverRipens = -1

// LongestRipening is how far ahead Ripens looks for the day a card face begun
// now is learned.
const LongestRipening = 10 * 365

// mostAnswers is how many answers a walk of one card face gives it before
// giving it up.
const mostAnswers = 1000

// Ripens is how many days of review a card face begun now needs before this
// preset counts it learned, when every day that card face falls due in answers
// it.
//
// It is one number for the whole material nobody has begun: those card faces
// all stand at the same nothing. A rule no such card face reaches is
// NeverRipens.
//
// A day of the week at none of the load schedules nothing, so it asks the card
// face nothing and counts for none of the days the pace divides by. A week
// carrying such a day needs as many days of review as its slowest day of the
// week does, so the pace holds for a card face begun on any of them.
func Ripens(by Scheduler, d Day, p Preset, now time.Time) int {
	s := Simulation{By: by, Day: d}
	from := d.StartOf(now)
	out := 0
	for range 7 {
		one := s.ripens(p, from)
		if one == NeverRipens {
			return NeverRipens
		}
		out = max(out, one)
		from = d.EndOf(from)
	}
	return out
}

// ripens is how many days of review a card face begun on this day needs.
func (s Simulation) ripens(p Preset, open time.Time) int {
	var c Schedule
	days := 0
	for range LongestRipening {
		ends := s.Day.EndOf(open)
		if p.Share(open.Weekday()) == 0 {
			open = ends
			continue
		}
		c = s.settles(c, open, ends, p)
		if p.Learned(c, ends) {
			return days
		}
		days++
		open = ends
	}
	return NeverRipens
}

// answers is where one showing leaves a card face, at the hour the day opens. A
// card face the day is not asking for stands where it is.
func (s Simulation) answers(c Schedule, open, ends time.Time, p Preset, on *DueByDay) Schedule {
	if c.Seen() && !c.Due.Before(ends) {
		return c
	}
	return s.step(c, open, p, on)
}

// settles is where a day of review leaves a card face when the day answers
// every showing it asks for: an answer that leaves the card falling due before
// the day closes is a card the day asks again, up to MostShowings.
//
// It is the day with no budget over it, which is the day the ripening of a card
// face is counted in.
func (s Simulation) settles(c Schedule, open, ends time.Time, p Preset) Schedule {
	for range MostShowings {
		if c.Seen() && !s.Day.Owed(c, open) {
			break
		}
		c = s.step(c, open, p, nil)
	}
	return c
}

// reaches reports whether a card face standing here is learned on the day the
// preset aims at, when every day of review from now to that day answers every
// showing it falls due for.
//
// Nothing paces it: a card face this does not get there is one no pace gets
// there, because no pace can give it more days than there are.
func (s Simulation) reaches(p Preset, c Schedule, open, by time.Time) bool {
	for range mostAnswers {
		if c.Seen() && !c.Due.Before(by) {
			break
		}
		if c.Seen() && !c.Due.Before(s.Day.EndOf(open)) {
			// Nothing is asked of it until the day its schedule falls in.
			open = s.Day.StartOf(c.Due)
		}
		ends := s.Day.EndOf(open)
		// A day of the week at none of the load asks it nothing, and the next
		// day of review picks it up.
		if p.Share(open.Weekday()) != 0 {
			c = s.settles(c, open, ends, p)
		}
		open = ends
	}
	return p.Learned(c, by)
}

// short is how many of these card faces cannot be learned by the day the preset
// aims at, whatever the pace, and how many of a material nobody has begun.
func (s Simulation) short(p Preset, cards []Schedule, unseen int, open time.Time) int {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	by := s.Day.Ending(p.By)
	out := 0
	for _, c := range cards {
		if !s.reaches(p, c, open, by) {
			out++
		}
	}
	if unseen > 0 && !s.reaches(p, Schedule{}, open, by) {
		out += unseen
	}
	return out
}

// step is where one projected answer leaves a card face.
//
// Both endings are worked out and weighed by how likely the card is to come
// back, which is the run's own assumption, so a projection follows one card
// down the middle of what it may do. The phase is the one a card that came back
// is left in.
func (s Simulation) step(c Schedule, at time.Time, p Preset, on *DueByDay) Schedule {
	if c.Seen() {
		c.Stability = math.Max(c.Stability, LeastStability)
	}
	if !c.Seen() {
		good := s.By.Next(c, at, Good)
		good.Due = p.Places(on, at, good.Due)
		return good
	}
	back := s.recalls(c, at)
	good, again := s.By.Endings(c, at)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(at).Seconds() + (1-back)*again.Due.Sub(at).Seconds()
	out.Due = p.Places(on, at, at.Add(time.Duration(away*float64(time.Second))))
	return out
}

// recalls is how likely a card face standing here is to come back at this
// instant, under the assumption this run was given.
func (s Simulation) recalls(c Schedule, at time.Time) float64 {
	if s.Recalls == nil {
		return AsModelled(c, at)
	}
	return s.Recalls(c, at)
}
