package projection

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
func GetDaysToLearn(scheduler Scheduler, d Day, p Preset, now time.Time) int {
	s := Simulation{By: scheduler, Day: d}
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
func (s Simulation) canReachGoal(p Preset, c Schedule, open, delta time.Time) bool {
	for range mostAnswers {
		if !c.IsNew() && !c.Due.Before(delta) {
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
	return p.IsLearned(c, delta)
}

// short is how many of these card faces cannot be learned by the day the preset
// aims at, whatever the pace, and how many of a material nobody has begun.
func (s Simulation) short(p Preset, cards []Schedule, unreviewed int, open time.Time) int {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	delta := s.Day.GetEndOfDate(p.By)
	out := 0
	for _, c := range cards {
		if !s.canReachGoal(p, c, open, delta) {
			out++
		}
	}
	if unreviewed > 0 && !s.canReachGoal(p, Schedule{}, open, delta) {
		out += unreviewed
	}
	return out
}

// step is where one projected answer leaves a card face.
//
// Both endings are worked out and weighed by how likely the card is to come
// back, which is the run's own assumption, so a projection follows one card
// down the middle of what it may do. The phase is the one a card that came back
// is left in.
func (s Simulation) step(c Schedule, pos time.Time, p Preset, on *DueByDay) Schedule {
	if !c.IsNew() {
		c.Stability = math.Max(c.Stability, LeastStability)
	}
	if c.IsNew() {
		good := s.By.Next(c, pos, Good)
		good.Due = p.AssignDueDay(on, pos, good.Due)
		return good
	}
	back := s.getRecallProbability(c, pos)
	good, again := s.By.Endings(c, pos)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(pos).Seconds() + (1-back)*again.Due.Sub(pos).Seconds()
	out.Due = p.AssignDueDay(on, pos, pos.Add(time.Duration(away*float64(time.Second))))
	return out
}

// getRecallProbability is how likely a card face standing here is to come back at this
// instant, under the assumption this run was given.
func (s Simulation) getRecallProbability(c Schedule, pos time.Time) float64 {
	if s.Recalls == nil {
		return GetModelledRecall(c, pos)
	}
	return s.Recalls(c, pos)
}

// simulationDay holds execution state for a day session.
type simulationDay struct {
	sim          Simulation
	preset       Preset
	days         int
	daysToLearn  int
	base         int
	today        int
	open         time.Time
	ends         time.Time
	admits       Allowance
	closed       BudgetNames
	used         time.Duration
	reply        int
	observed     int
	begun        int
	charged      int
	faced        int
	take         int
	next         int
	isDebtDone   bool
	isRepeatDone bool
	isNewDone    bool
	remaining    int
	cards        []Schedule
	showings     []int
	due          []int
	spare        []int
	again        []int
	falls        [][]int
	reckoned     *learnedCount
	on           *DueByDay
}

// createFalls maps initial schedules to their due day relative to the start day.
func createFalls(cards []Schedule, days int, on *DueByDay, base int) [][]int {
	falls := make([][]int, days)
	for i, c := range cards {
		on.Add(c.Due)
		if day := max(0, on.GetDayNumber(c.Due)-base); day < days {
			falls[day] = append(falls[day], i)
		}
	}
	return falls
}

// getAnswersMap returns a lookup set of days where retention is calculated.
func (s Simulation) getAnswersMap(days int) map[int]bool {
	answers := make(map[int]bool, len(s.Retains))
	for _, day := range s.Retains {
		if day >= 0 && day < days {
			answers[day] = true
		}
	}
	return answers
}

// getSortedCards returns schedules sorted by due timestamp.
func getSortedCards(schedules map[CardFaceID]Schedule) []Schedule {
	cards := make([]Schedule, 0, len(schedules))
	for _, one := range schedules {
		cards = append(cards, one)
	}
	slices.SortFunc(cards, compareByDue)
	return cards
}

// createProjection initializes projection statistics for the run.
func (s Simulation) createProjection(
	days int, schedules map[CardFaceID]Schedule, unreviewed int, now time.Time, p Preset,
) Projection {
	rule, _, _ := p.GetLearnedRule()
	learns := NeverLearns
	if rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}

	out := Projection{
		Days:   days,
		Faces:  len(schedules) + unreviewed,
		Seen:   len(schedules),
		Clears: NeverClears,
		Learns: learns,
	}
	if CountOverdue(s.Day, schedules, now) == 0 {
		out.Clears = 0
	}
	return out
}

// getDaysToLearnGoal returns days required to learn cards for date goals.
func (s Simulation) getDaysToLearnGoal(p Preset, now time.Time) int {
	if p.Goal == GoalDate {
		return GetDaysToLearn(s.By, s.Day, p, now)
	}
	return 0
}

// createSimulationDay initializes simulation day state across the projected days.
func (s Simulation) createSimulationDay(
	days, base int,
	open time.Time,
	p Preset,
	daysToLearn int,
	cards []Schedule,
	falls [][]int,
	on *DueByDay,
	reckoned *learnedCount,
	unreviewed int,
) *simulationDay {
	return &simulationDay{
		sim:         s,
		preset:      p,
		days:        days,
		daysToLearn: daysToLearn,
		base:        base,
		open:        open,
		cards:       cards,
		showings:    make([]int, len(cards)),
		falls:       falls,
		on:          on,
		reckoned:    reckoned,
		remaining:   unreviewed,
	}
}

// initLearned initializes learned card tracking for the projection.
func (s Simulation) initLearned(p Preset, cards []Schedule, out *Projection, open time.Time) *learnedCount {
	reckoned := newLearnedCount(p, cards, open)
	out.Learned = reckoned.count
	if out.Learns == NeverLearns && out.Learned == out.Faces {
		out.Learns = 0
	}
	return reckoned
}

// startDay prepares allowances, queues, and budgets for the given day.
func (d *simulationDay) startDay(today int) {
	d.today = today
	d.ends = d.sim.Day.GetEnd(d.open)
	d.used = 0
	d.reply = 0
	d.observed = 0
	d.begun = 0
	d.charged = 0
	d.faced = 0
	d.closed = BudgetNames{}

	removed := Spent{}
	if today == 0 {
		removed = d.sim.Spent
	}
	d.admits = d.preset.GetAllowance(d.sim.Day, d.open, removed, d.remaining, d.daysToLearn)
	if d.admits.IsPaused() {
		d.closed = d.closed.Add(ClosedPaused)
	}
	d.isDebtDone = d.admits.IsPaused()
	d.isRepeatDone = d.admits.IsPaused()
	d.isNewDone = d.admits.IsPaused()

	slices.SortFunc(d.falls[today], func(a, b int) int {
		return compareByDue(d.cards[a], d.cards[b])
	})
	d.spare = mergeByDue(d.spare[:0], d.due[d.take:], d.falls[today], d.cards)
	d.due, d.spare = d.spare, d.due
	d.take = 0
	d.next = 0
	clear(d.showings)
	d.again = d.again[:0]
}

// isOwed reports whether unreviewed debt cards remain for the day.
func (d *simulationDay) isOwed() bool {
	return !d.isDebtDone && d.take < len(d.due)
}

// hasNew reports whether unseen cards remain to introduce today.
func (d *simulationDay) hasNew() bool {
	return !d.isNewDone && d.remaining > 0
}

// isBacklog reports whether repeated cards remain for today.
func (d *simulationDay) isBacklog() bool {
	return !d.isRepeatDone && d.next < len(d.again)
}

// hasPendingCards reports whether any cards remain to review or introduce.
func (d *simulationDay) hasPendingCards() bool {
	return d.isOwed() || d.hasNew() || d.isBacklog()
}

// isDebtNext reports whether the next step should review an existing card.
func (d *simulationDay) isDebtNext() bool {
	owed := d.isOwed()
	hasNew := d.hasNew()
	repeat := !owed && !hasNew
	return repeat || d.admits.IsNextCardDebt(d.observed, d.begun, owed, hasNew)
}

// isRepeat reports whether all initial debt and new cards are exhausted.
func (d *simulationDay) isRepeat() bool {
	return !d.isOwed() && !d.hasNew()
}

// getCardIndex returns the card index for the next review step.
func (d *simulationDay) getCardIndex(repeat bool) int {
	if repeat {
		return d.again[d.next]
	}
	return d.due[d.take]
}

// stopDebt marks debt or repeat reviews complete for the day.
func (d *simulationDay) stopDebt(repeat bool) {
	if repeat {
		d.isRepeatDone = true
		return
	}
	d.isDebtDone = true
}

// queueNextShowing places a reviewed card in the repeat queue or future bucket.
func (d *simulationDay) queueNextShowing(c Schedule, pos int) {
	if d.sim.Day.IsOwed(c, d.open) && d.showings[pos] < MostShowings {
		d.again = append(d.again, pos)
		return
	}
	if day := max(d.today+1, d.on.GetDayNumber(c.Due)-d.base); day < d.days {
		d.falls[day] = append(d.falls[day], pos)
	}
}

// stepDebt processes one card review repetition or debt card.
func (d *simulationDay) stepDebt() {
	repeat := d.isRepeat()
	pos := d.getCardIndex(repeat)
	counted := d.preset.Counts.IsCharged(d.showings[pos] > 0)
	if counted && d.admits.Limits.Reviews != ClosedNothing && d.charged >= d.admits.Reviews {
		d.closed = d.closed.Add(d.admits.Limits.Reviews)
		d.stopDebt(repeat)
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.sim.Cost.Review > d.admits.Minutes {
		d.closed = d.closed.Add(d.admits.Limits.Minutes)
		d.stopDebt(repeat)
		return
	}
	if repeat {
		d.next++
	} else {
		d.take++
	}
	d.used += d.sim.Cost.Review
	d.observed++
	d.reply++
	if counted {
		d.charged++
	}
	if d.showings[pos] == 0 {
		d.faced++
	}
	d.showings[pos]++
	d.cards[pos] = d.sim.getScheduleAfterShowing(d.cards[pos], d.open, d.ends, d.preset, d.on)
	d.reckoned.countAnswer(pos, d.cards[pos], d.ends)
	d.queueNextShowing(d.cards[pos], pos)
}

// stepNew introduces one unseen card into the session.
func (d *simulationDay) stepNew() {
	if d.admits.Limits.New != ClosedNothing && d.begun >= d.admits.New {
		d.closed = d.closed.Add(d.admits.Limits.New)
		d.isNewDone = true
		return
	}
	if d.admits.Limits.Minutes != ClosedNothing && d.used+d.sim.Cost.New > d.admits.Minutes {
		d.closed = d.closed.Add(d.admits.Limits.Minutes)
		d.isNewDone = true
		return
	}
	d.used += d.sim.Cost.New
	d.begun++
	d.reply++
	d.faced++
	d.remaining--
	one := d.sim.getScheduleAfterShowing(Schedule{}, d.open, d.ends, d.preset, d.on)
	d.cards = append(d.cards, one)
	d.showings = append(d.showings, 1)
	d.reckoned.countBegun(one, d.ends)
	offset := len(d.cards) - 1
	d.queueNextShowing(one, offset)
}

// finishDay records daily metrics and carries forward remaining repeated cards.
func (d *simulationDay) finishDay(out *Projection, answerRetained bool, elapsed *time.Duration) {
	if d.today+1 < d.days {
		d.falls[d.today+1] = append(d.falls[d.today+1], d.again[d.next:]...)
	}

	out.Answered += d.reply
	*elapsed += d.used
	out.Load = append(out.Load, d.reply)
	out.Faced = append(out.Faced, d.faced)
	out.Spent = append(out.Spent, d.used)
	out.Admitted = append(out.Admitted, !d.admits.IsPaused())
	out.Closed = append(out.Closed, d.closed)

	stands, back := d.reckoned.countLearned(d.cards, d.ends, out.Faces, answerRetained)
	out.Through = append(out.Through, getLearnedShare(stands, out.Faces))
	if out.Learns == NeverLearns && stands == out.Faces {
		out.Learns = len(out.Load)
	}

	backlog := len(d.due) - d.take + len(d.again) - d.next
	out.Backlog = append(out.Backlog, backlog)
	if out.Clears == NeverClears && backlog == 0 {
		out.Clears = len(out.Load)
	}
	if answerRetained {
		out.Retained.add(d.today, back)
	}
	d.open = d.ends
}

// finalizeProjection calculates daily averages and remaining owed cards.
func (s Simulation) finalizeProjection(
	out *Projection, elapsed time.Duration, cards []Schedule, open time.Time,
) {
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

// simulateDaySession processes a single day.
func (s Simulation) simulateDaySession(
	d *simulationDay,
	today int,
	out *Projection,
	answers map[int]bool,
	elapsed *time.Duration,
) {
	d.startDay(today)
	for d.hasPendingCards() {
		if d.isDebtNext() {
			d.stepDebt()
			continue
		}
		d.stepNew()
	}
	d.finishDay(out, answers[today], elapsed)
}

// Run projects the card faces forward from now.
func (s Simulation) Run(
	ctx context.Context, now time.Time, p Preset, schedules map[CardFaceID]Schedule, unreviewed int,
) (Projection, error) {
	days := s.GetDurationDays()
	answers := s.getAnswersMap(days)
	cards := getSortedCards(schedules)

	out := s.createProjection(days, schedules, unreviewed, now, p)
	open := s.Day.GetStart(now)
	on := NewDueByDay(s.Day)
	base := on.GetDayNumber(open)
	falls := createFalls(cards, days, on, base)

	reckoned := s.initLearned(p, cards, &out, open)
	out.Short = s.short(p, cards, unreviewed, open)
	daysToLearn := s.getDaysToLearnGoal(p, now)

	d := s.createSimulationDay(days, base, open, p, daysToLearn, cards, falls, on, reckoned, unreviewed)
	var elapsed time.Duration

	for today := range days {
		if err := ctx.Err(); err != nil {
			return Projection{}, err
		}
		s.simulateDaySession(d, today, &out, answers, &elapsed)
	}

	out.Seen = len(d.cards)
	s.finalizeProjection(&out, elapsed, d.cards, d.open)
	return out, nil
}
