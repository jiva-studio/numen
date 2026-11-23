package projection

import (
	"context"
	"math"
	"slices"
	"time"
)

// Ahead is how many days a projection runs when no count is set.
const Ahead = 90

// LeastStability is the lower bound on the stability of a projected review card,
// in days, equal to one minute. The scheduler divides by stability, so a review
// card whose stability has decayed to zero after a long run of lapses produces
// unusable numbers, and every figure computed from it would be unusable too.
const LeastStability = 1.0 / (24 * 60)

// MostShowings is the maximum number of times one day of review shows a review
// card. An answer that does not return the card for several minutes schedules it
// later in the same day; a card that keeps landing in the same day is dropped and
// picked up by the following day.
const MostShowings = 8

// Simulation projects a preset forward over the days ahead: its review cards answered day after
// day, inside the budgets it keeps. The share of the load each day of the week carries scales
// what that day admits, and an even load moves cards onto the days carrying least. Both are
// the preset's, and the session reads them the same way.
type Simulation struct {
	By      Scheduler
	Rules   DayRules
	Day     Calendar
	Cost    Cost
	Days    int
	Retains []int
	Recalls RecallChance
	Spent   Spent
}

// GetDurationDays returns how many days this run walks, counting the day it opens
// as the first. A run with no count set walks Ahead days.
func (s Simulation) GetDurationDays() int {
	if s.Days <= 0 {
		return Ahead
	}
	return s.Days
}

// LongestLearning is how far ahead GetDaysToLearn looks for the day a review card begun
// now is learned.
const LongestLearning = 10 * 365

// mostAnswers is how many answers a walk of one review card gives it before
// giving it up.
const mostAnswers = 1000

// newNoCost, newNoSpent and newNoCard return the zero values of their types.

func newNoCost() Cost {
	// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Cost value."
	return Cost{New: 0, Review: 0}
}

// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Spent value."
func newNoSpent() Spent { return Spent{Answered: 0, New: 0, Reviews: 0, Took: 0} }

func newNoCard() Card {
	// gomutants:disable-next-line RETURN_ZERO reason="Struct literal already holds the zero Card value."
	return Card{Due: time.Time{}, Last: time.Time{}, Reps: 0, Lapses: 0, Stability: 0, Difficulty: 0, Phase: 0}
}

// GetDaysToLearn returns how many days of review a review card begun now needs before this
// preset counts it as learned, when every due day answers it. It is one number for the whole
// unbegun material. A rule no such card satisfies yields NeverLearns. A day of the week with
// none of the load shows nothing and is not counted, so the pace holds for a card begun on any
// day of the week.
func GetDaysToLearn(scheduler Scheduler, rules DayRules, d Calendar, p Settings, now time.Time) int {
	s := Simulation{
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Helper getDaysToLearn uses LongestLearning and does not read Simulation.Days."
		By: scheduler, Rules: rules, Day: d, Cost: newNoCost(), Days: 0, Retains: nil, Recalls: nil, Spent: newNoSpent(),
	}
	from := d.GetStart(now)
	// gomutants:disable-next-line INTEGER_DECREMENT reason="Every non-failing weekday returns a non-negative day count that supersedes -1 in max."
	out := 0
	// gomutants:disable-next-line INTEGER_INCREMENT reason="Seven consecutive days already cover every weekday in the periodic weekly load."
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

func (s Simulation) getDaysToLearn(p Settings, open time.Time) int {
	var c Card
	days := 0
	for range LongestLearning {
		ends := s.Day.GetEnd(open)
		if s.Rules.GetShare(p, open.Weekday()) == 0 {
			open = ends
			continue
		}
		c = s.getScheduleAfterDay(c, open, p)
		if s.By.IsLearned(p, c, ends) {
			return days
		}
		days++
		open = ends
	}
	return NeverLearns
}

func (s Simulation) getScheduleAfterShowing(c Card, open, ends time.Time, p Settings, on Load) Card {
	if !c.IsNew() && !c.Due.Before(ends) {
		return c
	}
	return s.step(c, open, p, on)
}

func (s Simulation) getScheduleAfterDay(c Card, open time.Time, p Settings) Card {
	for range MostShowings {
		if !c.IsNew() && !c.isDue(s.Day, open) {
			break
		}
		c = s.step(c, open, p, nil)
	}
	return c
}

func (s Simulation) canReachGoal(p Settings, c Card, open, delta time.Time) bool {
	for range mostAnswers {
		if !c.IsNew() && !c.Due.Before(delta) {
			break
		}
		if !c.IsNew() && !c.Due.Before(s.Day.GetEnd(open)) {
			// The card is not shown until the day its schedule falls in.
			open = s.Day.GetStart(c.Due)
		}
		ends := s.Day.GetEnd(open)
		// A day of the week with none of the load does not show the card, and the next
		// day of review picks it up.
		if s.Rules.GetShare(p, open.Weekday()) != 0 {
			c = s.getScheduleAfterDay(c, open, p)
		}
		open = ends
	}
	return s.By.IsLearned(p, c, delta)
}

func (s Simulation) countShort(p Settings, cards []Card, unreviewed int, open time.Time) int {
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
	if unreviewed > 0 && !s.canReachGoal(p, newNoCard(), open, delta) {
		out += unreviewed
	}
	return out
}

func (s Simulation) step(c Card, pos time.Time, p Settings, on Load) Card {
	if !c.IsNew() {
		c.Stability = math.Max(c.Stability, LeastStability)
	}
	if c.IsNew() {
		good, _ := s.By.GetEndings(c, pos)
		good.Due = placeCard(on, p, pos, good.Due)
		return good
	}
	back := s.getRecallProbability(c, pos)
	good, again := s.By.GetEndings(c, pos)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(pos).Seconds() + (1-back)*again.Due.Sub(pos).Seconds()
	out.Due = placeCard(on, p, pos, pos.Add(time.Duration(away*float64(time.Second))))
	return out
}

func placeCard(on Load, p Settings, at, due time.Time) time.Time {
	if on == nil {
		return due
	}
	return on.PlaceCard(p, at, due)
}

func (s Simulation) getRecallProbability(c Card, pos time.Time) float64 {
	if s.Recalls == nil {
		return s.By.GetRecall(c, pos)
	}
	return s.Recalls(c, pos)
}

func createFalls(cards []Card, days int, on Load, base int) [][]int {
	falls := make([][]int, days)
	for i, c := range cards {
		on.Add(c.Due)
		if day := max(0, on.GetDayNumber(c.Due)-base); day < days {
			falls[day] = append(falls[day], i)
		}
	}
	return falls
}

func (s Simulation) getAnswersMap(days int) map[int]bool {
	answers := make(map[int]bool, len(s.Retains))
	for _, day := range s.Retains {
		if day >= 0 && day < days {
			answers[day] = true
		}
	}
	return answers
}

func getSortedCards(cards []Card) []Card {
	out := slices.Clone(cards)
	slices.SortFunc(out, compareByDue)
	return out
}

func (s Simulation) createProjection(
	days int, cards []Card, unreviewed int, now time.Time, p Settings,
) Projection {
	learns := NeverLearns
	if p.Rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}

	out := Projection{
		Days:        days,
		ReviewsADay: 0,
		MinutesADay: 0,
		Retained:    RetentionByDay{on: nil},
		Answered:    0,
		ReviewCards: len(cards) + unreviewed,
		Seen:        len(cards),
		Due:         0,
		Clears:      NeverClears,
		Learned:     0,
		Learns:      learns,
		Short:       0,
		Load:        nil,
		Reviewed:    nil,
		Spent:       nil,
		Admitted:    nil,
		Closed:      nil,
		Backlog:     nil,
		Through:     nil,
	}
	if CountOverdue(s.Day, cards, now) == 0 {
		out.Clears = 0
	}
	return out
}

func (s Simulation) getDaysToLearnGoal(p Settings, now time.Time) int {
	if p.Goal == GoalDate {
		return GetDaysToLearn(s.By, s.Rules, s.Day, p, now)
	}
	return 0
}

func createEmptyAllowance() Allowance {
	return Allowance{
		New:      0,
		Reviews:  0,
		Minutes:  0,
		Limits:   Limits{New: ClosedNothing, Reviews: ClosedNothing, Minutes: ClosedNothing},
		Backlog:  0,
		IsPaused: false,
	}
}

func (s Simulation) finalizeProjection(
	out *Projection, elapsed time.Duration, cards []Card, open time.Time,
) {
	if admitted := out.CountAdmitted(); admitted > 0 {
		request := 0
		for _, one := range out.Reviewed {
			request += one
		}
		out.ReviewsADay = float64(request) / float64(admitted)
		out.MinutesADay = elapsed.Minutes() / float64(admitted)
	}
	for _, c := range cards {
		if c.Due.Before(open) {
			out.Due++
		}
	}
}

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

// Run walks the days ahead from now: the begun cards answered as they fall due and the unreviewed ones begun as the
// days admit them.
func (s Simulation) Run(
	ctx context.Context, now time.Time, p Settings, begun []Card, unreviewed int,
) (Projection, error) {
	days := s.GetDurationDays()
	answers := s.getAnswersMap(days)
	cards := getSortedCards(begun)

	out := s.createProjection(days, begun, unreviewed, now, p)
	open := s.Day.GetStart(now)
	on := s.By.NewLoad()
	base := on.GetDayNumber(open)
	falls := createFalls(cards, days, on, base)

	reckoned := s.initLearned(p, cards, &out, open)
	out.Short = s.countShort(p, cards, unreviewed, open)
	daysToLearn := s.getDaysToLearnGoal(p, now)

	d := s.createSimulationDay(simulationSetup{
		days: days, base: base, open: open, p: p, daysToLearn: daysToLearn, cards: cards,
		falls: falls, on: on, reckoned: reckoned, unreviewed: unreviewed,
	})
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
