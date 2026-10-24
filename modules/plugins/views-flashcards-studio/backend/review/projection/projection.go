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
	// ReviewsADay and MinutesADay are the daily load, over the days the preset
	// admitted: the card faces a day asks, and how long its showings take.
	ReviewsADay float64
	MinutesADay float64
	// Retained is the share of the material that comes back, on the days the run
	// was asked to answer for.
	Retained RetentionByDay
	// Answered is how many answers were given over the days projected.
	Answered int
	// Faces is the material: every card face the preset schedules. Seen is how
	// many of them have been answered at least once by the last day.
	Faces int
	Seen  int
	// Owed is the card faces answered before and standing owed on the last day,
	// which is the backlog the budget did not carry.
	Owed int
	// Clears is how many days of review it takes before nothing is overdue: no
	// card face is left whose day has come and gone. A day that begins with
	// nothing overdue clears in none, and a pace that never gets there is
	// NeverClears.
	Clears int
	// Learned is how many card faces the preset counts as learned as the run
	// opens, under the rule the preset names.
	Learned int
	// Learns is how many days of review it takes before every card face the
	// preset schedules is learned. A run opening with all of them learned learns
	// in none, and a horizon ending with one of them still to learn is
	// NeverLearns.
	//
	// It is answered where that day is a day. An interval is passed once and
	// stays passed, so the day the last card face passes it is one. A chance of
	// recall falls as a card fades and rises when it is answered, so a material
	// aiming at a date is answered by its date. Both are LearnsUnasked.
	Learns int
	// Short is how many card faces cannot be learned by the day the preset aims
	// at, whatever the pace: the rule wants more days than the date leaves them.
	// A preset aiming at no day has none.
	Short int
	// Load is how many showings each day projected carried, Faced is how many
	// card faces those showings were of, and Spent is how long they took.
	//
	// A day asks a card face again while its answer leaves it falling due before
	// the day closes. A budget kept in cards is spent on Faced and one kept in
	// showings on Load, and the minutes go on every showing either way.
	Load  []int
	Faced []int
	Spent []time.Duration
	// Admitted is whether the preset admitted each day projected. A day it did
	// not is no session at all, and the summaries over the run pass over it.
	Admitted []bool
	// Closed is every budget that stopped each day projected asking for more,
	// one entry a day.
	Closed []BudgetNames
	// Backlog is how many card faces stood overdue at the end of each day
	// projected: their day had passed and that day did not get to them. It is
	// the pile a person watches shrink, and it begins where Overdue stands now.
	Backlog []int
	// Through is the share of the material learned by the end of each day
	// projected, under the rule the preset names. Getting through the material
	// is learning it, and there is no second reckoning of it.
	Through []float64
}

// RetentionByDay is the share of the material that comes back at the end of a
// day, on the days a run was asked to answer for. A day it was not asked for
// holds no share, and GetShare says so.
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

// isAllLearned reports whether every card face that can be learned by this day of
// the run stands learned on it. Short is how many cannot be, whatever the pace.
func (p Projection) isAllLearned(day int) bool {
	if p.Faces == 0 {
		return true
	}
	return p.Through[day] >= float64(p.Faces-p.Short)/float64(p.Faces)
}

// Session is the first day of this run the preset admits: the next session a
// person will actually sit down to. False is a run admitting no day at all,
// which holds no session.
//
// It is one real day of the run, so the count read off it is the count a
// session on that day hands a person. A day the preset does not admit is no
// session, and the day after it is the one a person meets.
func (p Projection) Session() (int, bool) {
	for day, admitted := range p.Admitted {
		if admitted {
			return day, true
		}
	}
	return 0, false
}

// CountOverdue is how many card faces standing at these schedules have had their day
// and were not answered on it.
//
// A card falling due later in the day holding at is not overdue: its day is
// has had no day.
func CountOverdue(d Day, schedules map[CardFaceID]Schedule, now time.Time) int {
	startTime := d.GetStart(now)
	out := 0
	for _, s := range schedules {
		if !s.IsNew() && s.Due.Before(startTime) {
			out++
		}
	}
	return out
}

// mergeByDue puts what a day before did not reach and what falls due in this one
// into one run, the card face owed longest first.
//
// Both are in that order already, so a day sees the whole of what it owes for
// the cost of what it owes.
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

// learnedCount is how much of the material stands learned and how much of it
// comes back, at the close of a day.
//
// Under a rule of an interval what stands learned is a fact about a card face's
// schedule: the count is carried from day to day and asked again only of a card
// face the day answered. Under a rule of a chance of recall it is a fact about
// the instant, and is read off the same number as the share that comes back.
type learnedCount struct {
	preset    Preset
	isCarried bool
	target    float64
	learned   []bool
	count     int
}

// newLearnedCount opens the count over the card faces a run begins with, at the
// instant it opens on.
func newLearnedCount(p Preset, cards []Schedule, pos time.Time) *learnedCount {
	rule, _, retention := p.GetLearnedRule()
	out := &learnedCount{preset: p, isCarried: rule == RuleInterval, target: retention}
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

// countAnswer carries one card face the day has answered.
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

// countBegun carries one card face the day has begun.
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

// finish is how many card faces stand learned at this instant, and what share
// of the material comes back at it where the day is one the run answers for. A
// card face nobody has begun comes back to nobody, and counts in the material.
//
// A carried count is a fact about a card face's schedule, so the walk over the
// whole material is made on the days the share is wanted. A count read off the
// chance of recall is the same walk, and the share falls out of it.
func (r *learnedCount) countLearned(
	cards []Schedule, pos time.Time, faces int, wanted bool,
) (int, float64) {
	if r.isCarried {
		if !wanted || faces == 0 {
			return r.count, 0
		}
		back := 0.0
		for _, c := range cards {
			back += Recall(pos.Sub(c.Last), c.Stability)
		}
		return r.count, back / float64(faces)
	}
	stands, back := 0, 0.0
	for _, c := range cards {
		one := Recall(pos.Sub(c.Last), c.Stability)
		back += one
		if !c.IsNew() && one >= r.target {
			stands++
		}
	}
	if faces == 0 {
		return stands, 0
	}
	return stands, back / float64(faces)
}

// compareByDue puts the card face owed longest first. Two schedules alike in all of
// this are one card face as far as any of these numbers go.
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
func getLearnedShare(learned, faces int) float64 {
	if faces == 0 {
		return 1
	}
	return float64(learned) / float64(faces)
}
