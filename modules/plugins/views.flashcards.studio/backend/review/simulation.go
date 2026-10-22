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
// getRetainedDays returns a lookup set of days within the duration whose retention is evaluated.
func getRetainedDays(retains []int, totalDays int) map[int]bool {
	answers := make(map[int]bool, len(retains))
	for _, day := range retains {
		if day >= 0 && day < totalDays {
			answers[day] = true
		}
	}
	return answers
}

// getSortedCards returns the schedule cards sorted by due date.
func getSortedCards(at map[CardFaceID]Schedule) []Schedule {
	cards := make([]Schedule, 0, len(at))
	for _, one := range at {
		cards = append(cards, one)
	}
	slices.SortFunc(cards, compareByDue)
	return cards
}

// createInitialProjection initializes the projection structure with defaults and overdue status.
func createInitialProjection(p Preset, totalDays, totalCards, unseen int, overdue int) Projection {
	rule, _, _ := p.getLearnedRule()
	learns := NeverLearns
	if rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}
	out := Projection{
		Days:   totalDays,
		Faces:  totalCards + unseen,
		Seen:   totalCards,
		Clears: NeverClears,
		Learns: learns,
	}
	if overdue == 0 {
		out.Clears = 0
	}
	return out
}

// createInitialFalls distributes cards into buckets according to their due date.
func createInitialFalls(cards []Schedule, on *DueByDay, base, days int) [][]int {
	falls := make([][]int, days)
	for i, c := range cards {
		on.Add(c.Due)
		if day := max(0, on.number(c.Due)-base); day < days {
			falls[day] = append(falls[day], i)
		}
	}
	return falls
}

// finalizeProjection computes average review and minute rates and counts overdue cards.
func finalizeProjection(out *Projection, cards []Schedule, elapsed time.Duration, open time.Time) {
	if admitted := out.CountAdmitted(); admitted > 0 {
		request := 0
		for _, one := range out.Faced {
			request += one
		}
		out.ReviewsADay = float64(request) / float64(admitted)
		out.MinutesADay = elapsed.Minutes() / float64(admitted)
	}
	for _, c := range cards {
		if c.Due.Before(open) {
			out.Owed++
		}
	}
}

// recordDayMetrics records the outcome and progression of one simulated day into the projection.
func recordDayMetrics(
	out *Projection,
	cards []Schedule,
	ends time.Time,
	answersToday bool,
	reckoned *learnedCount,
	d *simulationDay,
) {
	out.Answered += d.reply
	out.Load = append(out.Load, d.reply)
	out.Faced = append(out.Faced, d.faced)
	out.Spent = append(out.Spent, d.used)
	out.Admitted = append(out.Admitted, !d.admits.IsPaused())
	out.Closed = append(out.Closed, d.closed)

	stands, back := reckoned.countLearned(cards, ends, out.Faces, answersToday)
	out.Through = append(out.Through, getLearnedShare(stands, out.Faces))
	if out.Learns == NeverLearns && stands == out.Faces {
		out.Learns = len(out.Load)
	}

	backlog := len(d.due) - d.take + len(d.again) - d.next
	out.Backlog = append(out.Backlog, backlog)
	if out.Clears == NeverClears && backlog == 0 {
		out.Clears = len(out.Load)
	}
	if answersToday {
		out.Retained.add(d.today, back)
	}
}

// canSimulateDebt reports whether the next card to review is an overdue card or repeated card.
func canSimulateDebt(repeat bool, admits Allowance, observed, begun int, owed, hasNew bool) bool {
	return repeat || admits.IsNextCardDebt(observed, begun, owed, hasNew)
}

// isReviewCountReached reports whether the allowance for reviews has been exhausted.
func isReviewCountReached(admits Allowance, charged int, counted bool) bool {
	return counted && admits.Limits.Reviews != ClosedNothing && charged >= admits.Reviews
}

// isReviewTimeReached reports whether the available review time has been exceeded.
func isReviewTimeReached(admits Allowance, used, cost time.Duration) bool {
	return admits.Limits.Minutes != ClosedNothing && used+cost > admits.Minutes
}

// isNewCountReached reports whether the allowance for new cards has been exhausted.
func isNewCountReached(admits Allowance, begun int) bool {
	return admits.Limits.New != ClosedNothing && begun >= admits.New
}

// isNewTimeReached reports whether the available time for new cards has been exceeded.
func isNewTimeReached(admits Allowance, used, cost time.Duration) bool {
	return admits.Limits.Minutes != ClosedNothing && used+cost > admits.Minutes
}

// getSpent returns the spent budget for the given day index.
func (s Simulation) getSpent(today int) Spent {
	if today == 0 {
		return s.Spent
	}
	return Spent{}
}

// getInitialClosed returns the initial closed status of the day.
func getInitialClosed(admits Allowance) BudgetNames {
	if admits.IsPaused() {
		return BudgetNames{ClosedPaused}
	}
	return BudgetNames{}
}

// simulationDay tracks the state of a single day during simulation.
type simulationDay struct {
	today  int
	days   int
	base   int
	open   time.Time
	ends   time.Time
	p      Preset
	admits Allowance

	due      []int
	take     int
	again    []int
	next     int
	showings []int

	used     time.Duration
	reply    int
	observed int
	begun    int
	charged  int
	faced    int

	paid     bool
	resolved bool
	all      bool
	closed   BudgetNames
}

func (d *simulationDay) closeReview(budget BudgetName, repeat bool) {
	d.closed = d.closed.add(budget)
	if repeat {
		d.resolved = true
		return
	}
	d.paid = true
}

// queueNextShowing places a reviewed card into either the same-day repetition queue or a future day bucket.
func (s Simulation) queueNextShowing(
	c Schedule, at int, count int, open time.Time,
	today, days, base int, on *DueByDay, falls [][]int, again *[]int,
) {
	if s.Day.IsOwed(c, open) && count < MostShowings {
		*again = append(*again, at)
		return
	}
	if day := max(today+1, on.number(c.Due)-base); day < days {
		falls[day] = append(falls[day], at)
	}
}

// stepDebt executes one review or repetition step for an existing card.
func (s Simulation) stepDebt(
	d *simulationDay, cards []Schedule, reckoned *learnedCount, on *DueByDay, falls [][]int, repeat bool,
) {
	at := 0
	if repeat {
		at = d.again[d.next]
	} else {
		at = d.due[d.take]
	}
	counted := d.p.Counts.IsCharged(d.showings[at] > 0)
	if isReviewCountReached(d.admits, d.charged, counted) {
		d.closeReview(d.admits.Limits.Reviews, repeat)
		return
	}
	if isReviewTimeReached(d.admits, d.used, s.Cost.Review) {
		d.closeReview(d.admits.Limits.Minutes, repeat)
		return
	}
	if repeat {
		d.next++
	} else {
		d.take++
	}
	d.used += s.Cost.Review
	d.observed++
	d.reply++
	if counted {
		d.charged++
	}
	if d.showings[at] == 0 {
		d.faced++
	}
	d.showings[at]++
	cards[at] = s.getScheduleAfterShowing(cards[at], d.open, d.ends, d.p, on)
	reckoned.countAnswer(at, cards[at], d.ends)
	s.queueNextShowing(cards[at], at, d.showings[at], d.open, d.today, d.days, d.base, on, falls, &d.again)
}

// stepNew introduces and schedules one unseen card face.
func (s Simulation) stepNew(
	d *simulationDay, cards *[]Schedule, reckoned *learnedCount, on *DueByDay, falls [][]int,
	remaining *int, out *Projection,
) {
	if isNewCountReached(d.admits, d.begun) {
		d.closed = d.closed.add(d.admits.Limits.New)
		d.all = true
		return
	}
	if isNewTimeReached(d.admits, d.used, s.Cost.New) {
		d.closed = d.closed.add(d.admits.Limits.Minutes)
		d.all = true
		return
	}
	d.used += s.Cost.New
	d.begun++
	d.reply++
	d.faced++
	*remaining--
	out.Seen++
	one := s.getScheduleAfterShowing(Schedule{}, d.open, d.ends, d.p, on)
	*cards = append(*cards, one)
	d.showings = append(d.showings, 1)
	reckoned.countBegun(one, d.ends)
	at := len(*cards) - 1
	s.queueNextShowing(one, at, d.showings[at], d.open, d.today, d.days, d.base, on, falls, &d.again)
}

// simulateDaySession processes all reviews and new card introductions for a single day.
func (s Simulation) simulateDaySession(
	d *simulationDay, cards *[]Schedule, reckoned *learnedCount, on *DueByDay, falls [][]int,
	remaining *int, out *Projection,
) {
	for {
		owed := !d.paid && d.take < len(d.due)
		hasNew := !d.all && *remaining > 0
		back := !d.resolved && d.next < len(d.again)
		if !owed && !hasNew && !back {
			break
		}
		repeat := !owed && !hasNew
		if canSimulateDebt(repeat, d.admits, d.observed, d.begun, owed, hasNew) {
			s.stepDebt(d, *cards, reckoned, on, falls, repeat)
			continue
		}
		s.stepNew(d, cards, reckoned, on, falls, remaining, out)
	}
}

// Run projects the card faces forward from now.
//
// The map is where the answers have left every card face the preset schedules,
// and unseen is how many of its card faces nobody has answered. A day's answers
// are all given at the hour the day opens, and the day asks a card face again
// while its answer leaves it falling due before the day closes.
func (s Simulation) initLearned(
	p Preset, cards []Schedule, open, now time.Time, unseen int, out *Projection,
) (*learnedCount, int) {
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
	return reckoned, daysToLearn
}

func (s Simulation) Run(
	ctx context.Context, now time.Time, p Preset, at map[CardFaceID]Schedule, unseen int,
) (Projection, error) {
	days := s.GetDurationDays()
	answers := getRetainedDays(s.Retains, days)
	cards := getSortedCards(at)
	out := createInitialProjection(p, days, len(cards), unseen, CountOverdue(s.Day, at, now))

	open := s.Day.GetStart(now)
	on := NewDueByDay(s.Day)
	base := on.number(open)
	falls := createInitialFalls(cards, on, base, days)

	reckoned, daysToLearn := s.initLearned(p, cards, open, now, unseen, &out)

	remaining := unseen
	var elapsed time.Duration
	var due, spare, again []int
	showings := make([]int, len(cards))
	take := 0

	for today := range days {
		if err := ctx.Err(); err != nil {
			return Projection{}, err
		}
		ends := s.Day.GetEnd(open)
		admits := p.GetAllowance(s.Day, open, s.getSpent(today), remaining, daysToLearn)

		slices.SortFunc(falls[today], func(a, b int) int { return compareByDue(cards[a], cards[b]) })
		spare = mergeByDue(spare[:0], due[take:], falls[today], cards)
		due, spare = spare, due
		take = 0
		if len(showings) < len(cards) {
			showings = make([]int, len(cards))
		} else {
			clear(showings)
		}
		again = again[:0]

		day := simulationDay{
			today:    today,
			days:     days,
			base:     base,
			open:     open,
			ends:     ends,
			p:        p,
			admits:   admits,
			due:      due,
			take:     take,
			again:    again,
			next:     0,
			showings: showings,
			paid:     admits.IsPaused(),
			resolved: admits.IsPaused(),
			all:      admits.IsPaused(),
			closed:   getInitialClosed(admits),
		}

		s.simulateDaySession(&day, &cards, reckoned, on, falls, &remaining, &out)

		if today+1 < days {
			falls[today+1] = append(falls[today+1], day.again[day.next:]...)
		}

		elapsed += day.used
		due = day.due
		take = day.take
		showings = day.showings

		recordDayMetrics(&out, cards, ends, answers[today], reckoned, &day)
		open = ends
	}

	finalizeProjection(&out, cards, elapsed, open)
	return out, nil
}
