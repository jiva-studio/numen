package review

import (
	"context"
	"math"
	"slices"
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
	// reads GetModelledRecall.
	Recalls RecallChance
	// Spent is what the day holding now has already gone through under this
	// preset. The first day of a run is a real day a person may be halfway
	// through, and what it has left is what a session opened now would offer.
	Spent Spent
}

// GetDurationDays is how many days this run walks, counting the day it opens as the
// first. A run told nothing walks Ahead of them.
func (s Simulation) GetDurationDays() int {
	if s.Days <= 0 {
		return Ahead
	}
	return s.Days
}

// LongestLearning is how far ahead GetDaysToLearn looks for the day a card face begun
// now is learned.
const LongestLearning = 10 * 365

// mostAnswers is how many answers a walk of one card face gives it before
// giving it up.
const mostAnswers = 1000

// GetDaysToLearn is how many days of review a card face begun now needs before this
// preset counts it learned, when every day that card face falls due in answers
// it.
//
// It is one number for the whole material nobody has begun: those card faces
// all stand at the same nothing. A rule no such card face reaches is
// NeverLearns.
//
// A day of the week at none of the load schedules nothing, so it asks the card
// face nothing and counts for none of the days the pace divides by. A week
// carrying such a day needs as many days of review as its slowest day of the
// week does, so the pace holds for a card face begun on any of them.
func GetDaysToLearn(by Scheduler, d Day, p Preset, now time.Time) int {
	s := Simulation{By: by, Day: d}
	from := d.GetStart(now)
	out := 0
	for range 7 {
		one := s.getDaysToLearn(p, from)
		if one == NeverLearns {
			return NeverLearns
		}
		out = max(out, one)
		from = d.GetEnd(from)
	}
	return out
}

// getDaysToLearn is how many days of review a card face begun on this day needs.
func (s Simulation) getDaysToLearn(p Preset, open time.Time) int {
	var c Schedule
	days := 0
	for range LongestLearning {
		ends := s.Day.GetEnd(open)
		if p.GetShare(open.Weekday()) == 0 {
			open = ends
			continue
		}
		c = s.getScheduleAfterDay(c, open, p)
		if p.IsLearned(c, ends) {
			return days
		}
		days++
		open = ends
	}
	return NeverLearns
}

// getScheduleAfterShowing is where one showing leaves a card face, at the hour the day
// opens. A card face the day is not asking for stands where it is.
func (s Simulation) getScheduleAfterShowing(c Schedule, open, ends time.Time, p Preset, on *DueByDay) Schedule {
	if !c.IsNew() && !c.Due.Before(ends) {
		return c
	}
	return s.step(c, open, p, on)
}

// getScheduleAfterDay is where a day of review leaves a card face when the day answers
// every showing it asks for: an answer that leaves the card falling due before
// the day closes is a card the day asks again, up to MostShowings.
//
// It is the day with no budget over it, which is the day the learning of a card
// face is counted in.
func (s Simulation) getScheduleAfterDay(c Schedule, open time.Time, p Preset) Schedule {
	for range MostShowings {
		if !c.IsNew() && !s.Day.IsOwed(c, open) {
			break
		}
		c = s.step(c, open, p, nil)
	}
	return c
}

// canReachGoal reports whether a card face standing here is learned on the day the
// preset aims at, when every day of review from now to that day answers every
// showing it falls due for.
//
// Nothing paces it: a card face this does not get there is one no pace gets
// there, because no pace can give it more days than there are.
func (s Simulation) canReachGoal(p Preset, c Schedule, open, by time.Time) bool {
	for range mostAnswers {
		if !c.IsNew() && !c.Due.Before(by) {
			break
		}
		if !c.IsNew() && !c.Due.Before(s.Day.GetEnd(open)) {
			// Nothing is asked of it until the day its schedule falls in.
			open = s.Day.GetStart(c.Due)
		}
		ends := s.Day.GetEnd(open)
		// A day of the week at none of the load asks it nothing, and the next
		// day of review picks it up.
		if p.GetShare(open.Weekday()) != 0 {
			c = s.getScheduleAfterDay(c, open, p)
		}
		open = ends
	}
	return p.IsLearned(c, by)
}

// short is how many of these card faces cannot be learned by the day the preset
// aims at, whatever the pace, and how many of a material nobody has begun.
func (s Simulation) short(p Preset, cards []Schedule, unseen int, open time.Time) int {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	by := s.Day.GetEndOfDate(p.By)
	out := 0
	for _, c := range cards {
		if !s.canReachGoal(p, c, open, by) {
			out++
		}
	}
	if unseen > 0 && !s.canReachGoal(p, Schedule{}, open, by) {
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
	if !c.IsNew() {
		c.Stability = math.Max(c.Stability, LeastStability)
	}
	if c.IsNew() {
		good := s.By.Next(c, at, Good)
		good.Due = p.AssignDueDay(on, at, good.Due)
		return good
	}
	back := s.getRecallProbability(c, at)
	good, again := s.By.Endings(c, at)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(at).Seconds() + (1-back)*again.Due.Sub(at).Seconds()
	out.Due = p.AssignDueDay(on, at, at.Add(time.Duration(away*float64(time.Second))))
	return out
}

// getRecallProbability is how likely a card face standing here is to come back at this
// instant, under the assumption this run was given.
func (s Simulation) getRecallProbability(c Schedule, at time.Time) float64 {
	if s.Recalls == nil {
		return GetModelledRecall(c, at)
	}
	return s.Recalls(c, at)
}

// Run projects the card faces forward from now.
//
// The map is where the answers have left every card face the preset schedules,
// and unseen is how many of its card faces nobody has answered. A day's answers
// are all given at the hour the day opens, and the day asks a card face again
// while its answer leaves it falling due before the day closes.
//
// A run over many days is long enough that a caller may give up on it, so the
// day it is on is where it is left.
func (s Simulation) Run(
	ctx context.Context, now time.Time, p Preset, at map[CardFaceID]Schedule, unseen int,
) (Projection, error) {
	r := newSimulationRun(s, now, p, at, unseen)
	open := r.open
	for today := range r.days {
		if err := ctx.Err(); err != nil {
			return Projection{}, err
		}
		ends := s.Day.GetEnd(open)
		d := r.newSimulationDay(today, open, ends)
		d.step()
		r.recordDay(d)
		open = ends
	}
	r.open = open
	return r.finish(), nil
}

type simulationRun struct {
	sim         Simulation
	preset      Preset
	now         time.Time
	open        time.Time
	base        int
	days        int
	daysToLearn int
	answers     map[int]bool
	cards       []Schedule
	falls       [][]int
	on          *DueByDay
	reckoned    *learnedCount
	unseen      int
	out         Projection
	spent       time.Duration

	due   []int
	spare []int
	take  int
	shown []int
	again []int
}

func newSimulationRun(
	s Simulation, now time.Time, p Preset, at map[CardFaceID]Schedule, unseen int,
) *simulationRun {
	days := s.GetDurationDays()
	answers := make(map[int]bool, len(s.Retains))
	for _, day := range s.Retains {
		if day >= 0 && day < days {
			answers[day] = true
		}
	}

	cards := make([]Schedule, 0, len(at))
	for _, one := range at {
		cards = append(cards, one)
	}
	slices.SortFunc(cards, compareByDue)

	rule, _, _ := p.getLearnedRule()
	learns := NeverLearns
	if rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}

	out := Projection{
		Days: days, Faces: len(at) + unseen, Seen: len(at),
		Clears: NeverClears, Learns: learns,
	}
	if CountOverdue(s.Day, at, now) == 0 {
		out.Clears = 0
	}

	open := s.Day.GetStart(now)
	on := NewDueByDay(s.Day)
	base := on.number(open)
	falls := make([][]int, days)
	for i, c := range cards {
		on.Add(c.Due)
		if day := max(0, on.number(c.Due)-base); day < days {
			falls[day] = append(falls[day], i)
		}
	}
	reckoned := newLearnedCount(p, cards, open)
	out.Learned = reckoned.count
	if out.Learns == NeverLearns && out.Learned == out.Faces {
		out.Learns = 0
	}
	out.Short = s.short(p, cards, unseen, open)
	daysToLearn := 0
	if p.Goal == GoalDate {
		daysToLearn = GetDaysToLearn(s.By, s.Day, p, now)
	}

	return &simulationRun{
		sim:         s,
		preset:      p,
		now:         now,
		open:        open,
		base:        base,
		days:        days,
		daysToLearn: daysToLearn,
		answers:     answers,
		cards:       cards,
		falls:       falls,
		on:          on,
		reckoned:    reckoned,
		unseen:      unseen,
		out:         out,
		shown:       make([]int, len(cards)),
	}
}

func (r *simulationRun) newSimulationDay(today int, open, ends time.Time) *simulationDay {
	gone := Spent{}
	if today == 0 {
		gone = r.sim.Spent
	}
	admits := r.preset.GetAllowance(r.sim.Day, open, gone, r.unseen, r.daysToLearn)

	slices.SortFunc(r.falls[today], func(a, b int) int {
		return compareByDue(r.cards[a], r.cards[b])
	})
	r.spare = mergeByDue(r.spare[:0], r.due[r.take:], r.falls[today], r.cards)
	r.due, r.spare = r.spare, r.due
	r.take = 0
	clear(r.shown)
	r.again = r.again[:0]

	var closed BudgetNames
	if admits.IsPaused() {
		closed = closed.add(ClosedPaused)
	}

	return &simulationDay{
		run:     r,
		today:   today,
		open:    open,
		ends:    ends,
		admits:  admits,
		paid:    admits.IsPaused(),
		settled: admits.IsPaused(),
		all:     admits.IsPaused(),
		closed:  closed,
	}
}

func (r *simulationRun) recordDay(d *simulationDay) {
	if d.today+1 < r.days {
		r.falls[d.today+1] = append(r.falls[d.today+1], r.again[d.next:]...)
	}

	r.out.Answered += d.answered
	r.spent += d.used
	r.out.Load = append(r.out.Load, d.answered)
	r.out.Faced = append(r.out.Faced, d.faced)
	r.out.Spent = append(r.out.Spent, d.used)
	r.out.Admitted = append(r.out.Admitted, !d.admits.IsPaused())
	r.out.Closed = append(r.out.Closed, d.closed)

	stands, back := r.reckoned.countLearned(r.cards, d.ends, r.out.Faces, r.answers[d.today])
	r.out.Through = append(r.out.Through, getLearnedShare(stands, r.out.Faces))
	if r.out.Learns == NeverLearns && stands == r.out.Faces {
		r.out.Learns = len(r.out.Load)
	}

	backlog := len(r.due) - r.take + len(r.again) - d.next
	r.out.Backlog = append(r.out.Backlog, backlog)
	if r.out.Clears == NeverClears && backlog == 0 {
		r.out.Clears = len(r.out.Load)
	}
	if r.answers[d.today] {
		r.out.Retained.add(d.today, back)
	}
}

func (r *simulationRun) finish() Projection {
	if admitted := r.out.CountAdmitted(); admitted > 0 {
		asked := 0
		for _, one := range r.out.Faced {
			asked += one
		}
		r.out.ReviewsADay = float64(asked) / float64(admitted)
		r.out.MinutesADay = r.spent.Minutes() / float64(admitted)
	}
	for _, c := range r.cards {
		if c.Due.Before(r.open) {
			r.out.Owed++
		}
	}
	return r.out
}

type simulationDay struct {
	run      *simulationRun
	today    int
	open     time.Time
	ends     time.Time
	admits   Allowance
	used     time.Duration
	answered int
	seen     int
	begun    int
	charged  int
	faced    int
	paid     bool
	settled  bool
	all      bool
	closed   BudgetNames
	next     int
}

func (d *simulationDay) step() {
	for {
		owed := !d.paid && d.run.take < len(d.run.due)
		fresh := !d.all && d.run.unseen > 0
		back := !d.settled && d.next < len(d.run.again)
		if !owed && !fresh && !back {
			break
		}
		repeat := !owed && !fresh
		if repeat || d.admits.IsNextCardDebt(d.seen, d.begun, owed, fresh) {
			d.answerReview(repeat)
		} else {
			d.answerNew()
		}
	}
}

func (d *simulationDay) answerReview(repeat bool) {
	at := 0
	if repeat {
		at = d.run.again[d.next]
	} else {
		at = d.run.due[d.run.take]
	}
	counted := d.run.preset.Counts.IsCharged(d.run.shown[at] > 0)
	if counted && d.admits.Limits.Reviews != ClosedNothing && d.charged >= d.admits.Reviews {
		d.closed = d.closed.add(d.admits.Limits.Reviews)
		d.closeReviewBudget(repeat)
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.run.sim.Cost.Review > d.admits.Minutes {
		d.closed = d.closed.add(d.admits.Limits.Minutes)
		d.closeReviewBudget(repeat)
		return
	}
	if repeat {
		d.next++
	} else {
		d.run.take++
	}
	d.used += d.run.sim.Cost.Review
	d.seen++
	d.answered++
	if counted {
		d.charged++
	}
	if d.run.shown[at] == 0 {
		d.faced++
	}
	d.run.shown[at]++
	d.run.cards[at] = d.run.sim.getScheduleAfterShowing(d.run.cards[at], d.open, d.ends, d.run.preset, d.run.on)
	d.run.reckoned.countAnswer(at, d.run.cards[at], d.ends)
	d.scheduleNextShowing(at)
}

func (d *simulationDay) closeReviewBudget(repeat bool) {
	if repeat {
		d.settled = true
	} else {
		d.paid = true
	}
}

func (d *simulationDay) answerNew() {
	if d.admits.Limits.New != ClosedNothing && d.begun >= d.admits.New {
		d.closed, d.all = d.closed.add(d.admits.Limits.New), true
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.run.sim.Cost.New > d.admits.Minutes {
		d.closed, d.all = d.closed.add(d.admits.Limits.Minutes), true
		return
	}
	d.used += d.run.sim.Cost.New
	d.begun++
	d.answered++
	d.faced++
	d.run.unseen--
	d.run.out.Seen++
	one := d.run.sim.getScheduleAfterShowing(Schedule{}, d.open, d.ends, d.run.preset, d.run.on)
	d.run.cards = append(d.run.cards, one)
	d.run.shown = append(d.run.shown, 1)
	d.run.reckoned.countBegun(one, d.ends)
	at := len(d.run.cards) - 1
	d.scheduleNextShowing(at)
}

func (d *simulationDay) scheduleNextShowing(cardIndex int) {
	if d.run.sim.Day.IsOwed(d.run.cards[cardIndex], d.open) && d.run.shown[cardIndex] < MostShowings {
		d.run.again = append(d.run.again, cardIndex)
		return
	}
	if day := max(d.today+1, d.run.on.number(d.run.cards[cardIndex].Due)-d.run.base); day < d.run.days {
		d.run.falls[day] = append(d.run.falls[day], cardIndex)
	}
}
