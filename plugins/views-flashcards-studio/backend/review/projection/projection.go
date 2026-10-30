package projection

import (
	"cmp"
	"maps"
	"slices"
	"time"
)

// Projection is what a preset evaluates to over the days ahead.
//
// Everything in it is a number a caller shows. It is worked out from the
// schedules the answers have already produced, and nothing in it is written
// anywhere.
type Projection struct {
	// Days is how many days were projected.
	Days int
	// ReviewsADay and MinutesADay are the daily load over the days the preset
	// admitted: the review cards a day returns, and how long its showings take.
	ReviewsADay float64
	MinutesADay float64
	// Retained is the share of the material that comes back, on the days the run
	// reports retention for.
	Retained RetentionByDay
	// Answered is how many answers were given over the days projected.
	Answered int
	// ReviewCards is the material: every review card the preset schedules. Seen is how
	// many of them have been answered at least once by the last day.
	ReviewCards int
	Seen        int
	// Due is the number of review cards answered before and still due on the last
	// day, which is the backlog the budget did not cover.
	Due int
	// Clears is how many days of review it takes before nothing is overdue: no
	// review card is left whose day has come and gone. A day that begins with
	// nothing overdue clears in none, and a pace that never gets there is
	// NeverClears.
	Clears int
	// Learned is how many review cards the preset counts as learned as the run
	// opens, under the rule the preset names.
	Learned int
	// Learns is how many days of review it takes before every review card the
	// preset schedules is learned. A run opening with all of them learned learns
	// in none, and a horizon ending with one of them still to learn is
	// NeverLearns.
	//
	// It is answered where that day is a day. An interval is passed once and
	// stays passed, so the day the last review card passes it is one. A chance of
	// recall falls as a card fades and rises when it is answered, so a material
	// aiming at a date is answered by its date. Both are LearnsUnasked.
	Learns int
	// Short is how many review cards cannot be learned by the target day of the
	// preset, at any pace: the rule needs more days than the date allows. A preset
	// with no target day has none.
	Short int
	// Load is how many showings each projected day contained, Reviewed is how many
	// review cards those showings were of, and Spent is how long they took.
	//
	// A day returns a review card again while its answer makes it due before the day
	// closes. A budget counted in cards is spent on Reviewed and one counted in
	// showings on Load, and the minutes are charged on every showing in both cases.
	Load     []int
	Reviewed []int
	Spent    []time.Duration
	// Admitted is whether the preset admitted each day projected. A day it did
	// not is no session at all, and the summaries over the run pass over it.
	Admitted []bool
	// Closed is every budget that stopped each day projected asking for more,
	// one entry a day.
	Closed []BudgetNames
	// Backlog is how many review cards are overdue at the end of each projected day:
	// their day has passed and that day did not reach them. It starts from the
	// current Overdue count and shrinks as the backlog clears.
	Backlog []int
	// Through is the share of the material learned by the end of each day
	// projected, under the rule the preset names. Getting through the material
	// is learning it, and there is no second reckoning of it.
	Through []float64
}

// RetentionByDay is the share of the material that comes back at the end of a
// day, for the days a run reports retention for. A day without a report has no
// share, and GetShare reports that.
type RetentionByDay struct{ on map[int]float64 }

// GetShare is the share of the material that came back at the end of this day
// of the run, counting the day the run opens as none, and whether the run
// answers for that day.
func (r RetentionByDay) GetShare(day int) (float64, bool) {
	share, answers := r.on[day]
	return share, answers
}

// Days is every day this answers for, in order.
func (r RetentionByDay) Days() []int { return slices.Sorted(maps.Keys(r.on)) }

// add records the share one day came to.
func (r *RetentionByDay) add(day int, share float64) {
	if r.on == nil {
		r.on = make(map[int]float64)
	}
	r.on[day] = share
}

// CountAdmitted is how many of the days projected the preset admitted.
func (p Projection) CountAdmitted() int {
	out := 0
	for _, one := range p.Admitted {
		if one {
			out++
		}
	}
	return out
}

// minutesADay is MinutesADay over the days of the run up to and including one.
//
// A day the preset does not admit is no session at all and takes no part: a
// week of five days runs its minutes over five days.
func (p Projection) minutesADay(limitDay int) float64 {
	var all time.Duration
	days := 0
	for i, one := range p.Spent[:limitDay+1] {
		if !p.Admitted[i] {
			continue
		}
		all += one
		days++
	}
	if days == 0 {
		return 0
	}
	return all.Minutes() / float64(days)
}

// isAllLearned reports whether every review card that can be learned by this day of
// isAllLearned reports whether every review card that can be learned by this day
// of the run is learned on it. Short is how many cannot be learned at any pace.
func (p Projection) isAllLearned(day int) bool {
	if p.ReviewCards == 0 {
		return true
	}
	return p.Through[day] >= float64(p.ReviewCards-p.Short)/float64(p.ReviewCards)
}

// Session is the first day of this run that the preset admits: the next session
// the user starts. False means the run admits no day, so it has no session.
//
// It is one real day of the run, so the count read from it is the count a session
// on that day returns. A day the preset does not admit is not a session, and the
// following day is the next one the user starts.
func (p Projection) Session() (int, bool) {
	for day, admitted := range p.Admitted {
		if admitted {
			return day, true
		}
	}
	return 0, false
}

// CountOverdue is how many review cards standing at these schedules have had their day
// and were not answered on it.
//
// A card falling due later in the day holding at is not overdue: its day is
// has had no day.
func CountOverdue(d Day, schedules map[ReviewCardID]Schedule, now time.Time) int {
	startTime := d.GetStart(now)
	out := 0
	for _, s := range schedules {
		if !s.IsNew() && s.Due.Before(startTime) {
			out++
		}
	}
	return out
}

// mergeByDue merges the carried cards from the previous day and the cards that
// fall due in this day into one run, the longest overdue review card first.
//
// Both inputs are already in that order, so a day processes everything it has due
// in a single linear pass.
func mergeByDue(into, carried, fell []int, cards []Schedule) []int {
	a, b := 0, 0
	for a < len(carried) && b < len(fell) {
		if compareByDue(cards[carried[a]], cards[fell[b]]) <= 0 {
			into = append(into, carried[a])
			a++
			continue
		}
		into = append(into, fell[b])
		b++
	}
	into = append(into, carried[a:]...)
	return append(into, fell[b:]...)
}

// learnedCount tracks how much of the material is learned and how much of it
// comes back, at the close of a day.
//
// Under an interval rule, learned is a property of a review card's schedule: the
// count is carried from day to day and recomputed only for review cards that the
// day answered. Under a recall-chance rule, learned is a property of the instant,
// and is read from the same value as the share that comes back.
type learnedCount struct {
	preset    Preset
	isCarried bool
	target    float64
	learned   []bool
	count     int
}

// newLearnedCount opens the count over the review cards a run begins with, at the
// instant it opens on.
func newLearnedCount(p Preset, cards []Schedule, pos time.Time) *learnedCount {
	rule, _, retention := p.GetLearnedRule()
	out := &learnedCount{preset: p, isCarried: rule == RuleInterval, target: retention, learned: nil, count: 0}
	if out.isCarried {
		out.learned = make([]bool, len(cards))
	}
	for i, c := range cards {
		if !p.IsLearned(c, pos) {
			continue
		}
		out.count++
		if out.isCarried {
			out.learned[i] = true
		}
	}
	return out
}

// countAnswer carries one review card the day has answered.
func (r *learnedCount) countAnswer(card int, c Schedule, pos time.Time) {
	if !r.isCarried {
		return
	}
	isLearned := r.preset.IsLearned(c, pos)
	if isLearned == r.learned[card] {
		return
	}
	r.learned[card] = isLearned
	if isLearned {
		r.count++
		return
	}
	r.count--
}

// countBegun carries one review card the day has begun.
func (r *learnedCount) countBegun(c Schedule, pos time.Time) {
	if !r.isCarried {
		return
	}
	isLearned := r.preset.IsLearned(c, pos)
	r.learned = append(r.learned, isLearned)
	if isLearned {
		r.count++
	}
}

// countLearned returns how many review cards are learned at this instant, and the
// share of the material that comes back at it when the day is one the run reports
// retention for. A review card that has not begun contributes nothing to the share
// and counts in the material.
//
// A carried count is a property of a review card's schedule, so the walk over the
// whole material runs only on days where the share is requested. A count read from
// the recall chance is the same walk, and the share is a by-product.
func (r *learnedCount) countLearned(
	cards []Schedule, pos time.Time, reviewCards int, wanted bool,
) (int, float64) {
	if r.isCarried {
		if !wanted || reviewCards == 0 {
			return r.count, 0
		}
		back := 0.0
		for _, c := range cards {
			back += Recall(pos.Sub(c.Last), c.Stability)
		}
		return r.count, back / float64(reviewCards)
	}
	stands, back := 0, 0.0
	for _, c := range cards {
		one := Recall(pos.Sub(c.Last), c.Stability)
		back += one
		if !c.IsNew() && one >= r.target {
			stands++
		}
	}
	if reviewCards == 0 {
		return stands, 0
	}
	return stands, back / float64(reviewCards)
}

// compareByDue orders review cards by due time, the longest overdue first. Two
// schedules that are equal in all compared fields are one review card for these
// numbers.
func compareByDue(a, b Schedule) int {
	if !a.Due.Equal(b.Due) {
		return a.Due.Compare(b.Due)
	}
	if !a.Last.Equal(b.Last) {
		return a.Last.Compare(b.Last)
	}
	if a.Stability != b.Stability {
		return cmp.Compare(a.Stability, b.Stability)
	}
	return cmp.Compare(a.Difficulty, b.Difficulty)
}

// getLearnedShare is the share of the material learned. A preset scheduling
// nothing is through all of it.
func getLearnedShare(learned, reviewCards int) float64 {
	if reviewCards == 0 {
		return 1
	}
	return float64(learned) / float64(reviewCards)
}
