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
	Days        int
	ReviewsADay float64
	MinutesADay float64
	Retained    RetentionByDay
	Answered    int
	ReviewCards int
	Seen        int
	Due         int
	Clears      int
	Learned     int
	Learns      int
	Short       int
	Load        []int
	Reviewed    []int
	Spent       []time.Duration
	Admitted    []bool
	Closed      []BudgetNames
	Backlog     []int
	Through     []float64
}

// RetentionByDay is the share of the material that is recalled at the end of a
// day, for the days a run reports retention for. A day without a report has no
// share, and GetShare reports that.
type RetentionByDay struct{ on map[int]float64 }

// GetShare is the share of the material that was recalled at the end of this day
// of the run, counting the day the run opens as none, and whether the run
// answers for that day.
func (r RetentionByDay) GetShare(day int) (float64, bool) {
	share, answers := r.on[day]
	return share, answers
}

// Days is every day this answers for, in order.
func (r RetentionByDay) Days() []int { return slices.Sorted(maps.Keys(r.on)) }

func addRetention(r *RetentionByDay, day int, share float64) {
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

func (p Projection) isAllLearned(day int) bool {
	if p.ReviewCards == 0 {
		return true
	}
	return p.Through[day] >= float64(p.ReviewCards-p.Short)/float64(p.ReviewCards)
}

// Session is the first day of this run that the preset admits: the next session the user
// starts. False means the run admits no day. It is one real day of the run, so the count read
// from it is the count a session on that day returns; a day the preset does not admit is not a
// session.
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
// is recalled, at the close of a day.
//
// Under an interval rule, learned is a property of a review card's schedule: the
// count is carried from day to day and recomputed only for review cards that the
// day answered. Under a recall-chance rule, learned is a property of the instant,
// and is read from the same value as the share that is recalled.
type learnedCount struct {
	preset    Preset
	isCarried bool
	target    float64
	learned   []bool
	count     int
}

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

func (r *learnedCount) countBegunCards(c Schedule, pos time.Time) {
	if !r.isCarried {
		return
	}
	isLearned := r.preset.IsLearned(c, pos)
	r.learned = append(r.learned, isLearned)
	if isLearned {
		r.count++
	}
}

func (r *learnedCount) countLearnedCards(
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

func getLearnedShare(learned, reviewCards int) float64 {
	if reviewCards == 0 {
		return 1
	}
	return float64(learned) / float64(reviewCards)
}
