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
	days := s.GetDurationDays()
	// The days this run answers the returning share for.
	answers := make(map[int]bool, len(s.Retains))
	for _, day := range s.Retains {
		if day >= 0 && day < days {
			answers[day] = true
		}
	}

	// The map hands its schedules over in whatever order it holds them, and a
	// projection asked twice answers the same both times.
	cards := make([]Schedule, 0, len(at))
	for _, one := range at {
		cards = append(cards, one)
	}
	slices.SortFunc(cards, compareByDue)

	// The day the whole material stands learned is asked of a preset counting by
	// an interval and aiming at no date. The other two are answered by a level
	// and by their own date.
	rule, _, _ := p.getLearnedRule()
	learns := NeverLearns
	if rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}

	out := Projection{
		Days: days, Faces: len(at) + unseen, Seen: len(at),
		Clears: NeverClears, Learns: learns,
	}
	// A day that begins with nothing overdue has nothing to clear.
	if CountOverdue(s.Day, at, now) == 0 {
		out.Clears = 0
	}
	left := unseen
	var spent time.Duration

	open := s.Day.GetStart(now)
	// Where the answers so far have left every card face is what the days
	// ahead are loaded with, and which day of the run first asks for it. A card
	// face falling due past the run is asked for on none of them.
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
	// A run opening with the whole material learned has nothing left to learn.
	if out.Learns == NeverLearns && out.Learned == out.Faces {
		out.Learns = 0
	}
	// What no pace reaches, and how long a card face begun today takes to be
	// learned. No goal but a date reads either, and neither is asked for under
	// another.
	out.Short = s.short(p, cards, unseen, open)
	daysToLearn := 0
	if p.Goal == GoalDate {
		daysToLearn = GetDaysToLearn(s.By, s.Day, p, now)
	}
	// The card faces the day may answer, and how many of them it reached. Both
	// are carried from one day to the next: what a day did not reach is the
	// front of what the day after it owes.
	var due, spare []int
	take := 0
	// How many times the day has asked each card face, and the faces it has
	// still to ask again before it closes.
	shown := make([]int, len(cards))
	var again []int
	for today := range days {
		if err := ctx.Err(); err != nil {
			return Projection{}, err
		}
		ends := s.Day.GetEnd(open)
		var used time.Duration

		// What the day admits is the one answer, and it is the answer the
		// session of that day will be held to.
		// The first day of a run is the day holding now, which a person may be
		// halfway through. Every day after it opens unspent.
		gone := Spent{}
		if today == 0 {
			gone = s.Spent
		}
		admits := p.GetAllowance(s.Day, open, gone, left, daysToLearn)

		// What has fallen due by the close of the day, the oldest debt first, so
		// a day that cannot pay all of it leaves the cards least overdue
		// standing: what the day before did not reach, and what falls due in
		// this one.
		slices.SortFunc(falls[today], func(a, b int) int { return compareByDue(cards[a], cards[b]) })
		spare = mergeByDue(spare[:0], due[take:], falls[today], cards)
		due, spare = spare, due
		take = 0
		clear(shown)
		again = again[:0]
		next := 0

		// What closed the day is every budget that turned a card away. A day
		// that asked for every card there was is closed by nothing.
		var closed BudgetNames
		if admits.IsPaused() {
			closed = closed.add(ClosedPaused)
		}

		// The day is spent between the debt and the material it has not begun,
		// in the share the preset names. A side the day has no more room for is
		// done with, and the other goes on with what is left of the day.
		// Answered is every showing the day gave; charged is the slots of the
		// count they spent, which is the first showing of a face or every one of
		// them, as the preset counts.
		answered, seen, begun, charged, faced := 0, 0, 0, 0, 0
		// Paid, settled and all are the three sides the day is done with: the
		// debt it opened on, the cards it has answered into the day itself, and
		// the material it has not begun.
		paid, settled, all := admits.IsPaused(), admits.IsPaused(), admits.IsPaused()
		for {
			owed := !paid && take < len(due)
			fresh := !all && left > 0
			back := !settled && next < len(again)
			if !owed && !fresh && !back {
				break
			}

			// A day hands over everything it owes and everything it begins
			// before it comes back to a card it has already shown, which is the
			// order the sessions of that day put them in.
			repeat := !owed && !fresh
			if repeat || admits.IsNextCardDebt(seen, begun, owed, fresh) {
				at := 0
				if repeat {
					at = again[next]
				} else {
					at = due[take]
				}
				counted := p.Counts.IsCharged(shown[at] > 0)
				if counted && admits.Limits.Reviews != ClosedNothing && charged >= admits.Reviews {
					closed = closed.add(admits.Limits.Reviews)
					if repeat {
						settled = true
					} else {
						paid = true
					}
					continue
				}
				if admits.Limits.Minutes != ClosedNothing && used+s.Cost.Review > admits.Minutes {
					closed = closed.add(admits.Limits.Minutes)
					if repeat {
						settled = true
					} else {
						paid = true
					}
					continue
				}
				if repeat {
					next++
				} else {
					take++
				}
				used += s.Cost.Review
				seen++
				answered++
				if counted {
					charged++
				}
				if shown[at] == 0 {
					faced++
				}
				shown[at]++
				cards[at] = s.getScheduleAfterShowing(cards[at], open, ends, p, on)
				reckoned.countAnswer(at, cards[at], ends)
				if s.Day.IsOwed(cards[at], open) && shown[at] < MostShowings {
					again = append(again, at)
					continue
				}
				if day := max(today+1, on.number(cards[at].Due)-base); day < days {
					falls[day] = append(falls[day], at)
				}
				continue
			}

			if admits.Limits.New != ClosedNothing && begun >= admits.New {
				closed, all = closed.add(admits.Limits.New), true
				continue
			}
			if admits.Limits.Minutes != ClosedNothing && used+s.Cost.New > admits.Minutes {
				closed, all = closed.add(admits.Limits.Minutes), true
				continue
			}
			used += s.Cost.New
			begun++
			answered++
			faced++
			left--
			out.Seen++
			one := s.getScheduleAfterShowing(Schedule{}, open, ends, p, on)
			cards = append(cards, one)
			shown = append(shown, 1)
			reckoned.countBegun(one, ends)
			at := len(cards) - 1
			if s.Day.IsOwed(one, open) && shown[at] < MostShowings {
				again = append(again, at)
				continue
			}
			if day := max(today+1, on.number(one.Due)-base); day < days {
				falls[day] = append(falls[day], at)
			}
		}

		// A card face the day put down before it settled falls due in a day that
		// is over, so the day after it picks it up.
		if today+1 < days {
			falls[today+1] = append(falls[today+1], again[next:]...)
		}

		out.Answered += answered
		spent += used
		out.Load = append(out.Load, answered)
		out.Faced = append(out.Faced, faced)
		out.Spent = append(out.Spent, used)
		out.Admitted = append(out.Admitted, !admits.IsPaused())
		out.Closed = append(out.Closed, closed)

		// How much of the material stands learned at the close of the day, which
		// is how far through it the day leaves a person, and how much of it
		// comes back at that hour.
		stands, back := reckoned.countLearned(cards, ends, out.Faces, answers[today])
		out.Through = append(out.Through, getLearnedShare(stands, out.Faces))
		if out.Learns == NeverLearns && stands == out.Faces {
			out.Learns = len(out.Load)
		}

		// What the day left standing is what fell due in it and was not reached,
		// and the day the backlog is gone is the first day none is.
		backlog := len(due) - take + len(again) - next
		out.Backlog = append(out.Backlog, backlog)
		if out.Clears == NeverClears && backlog == 0 {
			out.Clears = len(out.Load)
		}
		if answers[today] {
			out.Retained.add(today, back)
		}
		open = ends
	}

	if admitted := out.CountAdmitted(); admitted > 0 {
		asked := 0
		for _, one := range out.Faced {
			asked += one
		}
		out.ReviewsADay = float64(asked) / float64(admitted)
		out.MinutesADay = spent.Minutes() / float64(admitted)
	}
	for _, c := range cards {
		if c.Due.Before(open) {
			out.Owed++
		}
	}
	return out, nil
}
