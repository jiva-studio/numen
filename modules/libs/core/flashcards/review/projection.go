package review

import (
	"cmp"
	"context"
	"maps"
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

// Projection is what a preset comes to over the days ahead.
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
	// counted that way stands at a level and reaches no such day; and a preset
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
	// not is no sitting at all, and the summaries over the run pass over it.
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
// holds no share, and On says so.
type RetentionByDay struct{ on map[int]float64 }

// On is the share of the material that came back at the end of this day of the
// run, counting the day the run opens as none, and whether the run answers for
// that day.
func (r RetentionByDay) On(day int) (float64, bool) {
	share, answers := r.on[day]
	return share, answers
}

// Days is every day this answers for, in order.
func (r RetentionByDay) Days() []int { return slices.Sorted(maps.Keys(r.on)) }

// holds the share one day came to.
func (r *RetentionByDay) holds(day int, share float64) {
	if r.on == nil {
		r.on = make(map[int]float64)
	}
	r.on[day] = share
}

// NeverClears is a pace that leaves something overdue on every day projected.
const NeverClears = -1

// NeverLearns is a horizon that ends with a card face still to learn. The day
// the last of them is learned is further off than the projection ran, and it is
// not worked out from what the run saw.
const NeverLearns = -1

// LearnsUnasked is a projection with no day on which the whole material stands
// learned: one counting by a chance of recall, which is a level and not a
// milestone, and one aiming at a date, which is that day.
const LearnsUnasked = -2

// Admits is how many of the days projected the preset admitted.
func (p Projection) Admits() int {
	out := 0
	for _, one := range p.Admitted {
		if one {
			out++
		}
	}
	return out
}

// Sitting is the first day of this run the preset admits: the next sitting a
// person will actually sit down to. False is a run admitting no day at all,
// which holds no sitting.
//
// It is one real day of the run, so the count read off it is the count a
// sitting on that day hands a person. A day the preset does not admit is no
// sitting, and the day after it is the one a person meets.
func (p Projection) Sitting() (int, bool) {
	for day, admitted := range p.Admitted {
		if admitted {
			return day, true
		}
	}
	return 0, false
}

// Overdue is how many card faces standing at these schedules have had their day
// and were not answered on it.
//
// A card falling due later in the day holding at is not overdue: its day is
// this one. A card face nobody has answered is not overdue either, because it
// has had no day.
func Overdue(d Day, at map[CardFaceID]Schedule, now time.Time) int {
	opened := d.Opens(now)
	out := 0
	for _, s := range at {
		if s.Seen() && s.Due.Before(opened) {
			out++
		}
	}
	return out
}

// Simulation projects a preset forward over the days ahead: its card faces
// answered day after day, inside the budgets it keeps.
//
// The share of the load each day of the week carries scales what that day
// admits, and an even load moves cards onto the days carrying least. Both are
// Preset's, and the sitting reads them the same way.
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
	// through, and what it has left is what a sitting opened now would offer.
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
	days := s.Covers()
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
	slices.SortFunc(cards, older)

	// The day the whole material stands learned is asked of a preset counting by
	// an interval and aiming at no date. The other two are answered by a level
	// and by their own date.
	rule, _, _ := p.counting()
	learns := NeverLearns
	if rule != RuleInterval || p.Goal == GoalDate {
		learns = LearnsUnasked
	}

	out := Projection{
		Days: days, Faces: len(at) + unseen, Seen: len(at),
		Clears: NeverClears, Learns: learns,
	}
	// A day that begins with nothing overdue has nothing to clear.
	if Overdue(s.Day, at, now) == 0 {
		out.Clears = 0
	}
	left := unseen
	var spent time.Duration

	open := s.Day.Opens(now)
	// Where the answers so far have left every card face is what the days
	// ahead are loaded with, and which day of the run first asks for it. A card
	// face falling due past the run is asked for on none of them.
	on := Spreading(s.Day)
	base := on.number(open)
	falls := make([][]int, days)
	for i, c := range cards {
		on.Holds(c.Due)
		if day := max(0, on.number(c.Due)-base); day < days {
			falls[day] = append(falls[day], i)
		}
	}
	reckoned := reckons(p, cards, open)
	out.Learned = reckoned.count
	// A run opening with the whole material learned has nothing left to learn.
	if out.Learns == NeverLearns && out.Learned == out.Faces {
		out.Learns = 0
	}
	// What no pace reaches, and how long a card face begun today takes to be
	// learned. No goal but a date reads either, and neither is asked for under
	// another.
	out.Short = s.short(p, cards, unseen, open)
	ripens := 0
	if p.Goal == GoalDate {
		ripens = Ripens(s.By, s.Day, p, now)
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
		ends := s.Day.Ends(open)
		var used time.Duration

		// What the day admits is the one answer, and it is the answer the
		// sitting of that day will be held to.
		// The first day of a run is the day holding now, which a person may be
		// halfway through. Every day after it opens unspent.
		gone := Spent{}
		if today == 0 {
			gone = s.Spent
		}
		admits := p.Admits(s.Day, open, gone, left, ripens)

		// What has fallen due by the close of the day, the oldest debt first, so
		// a day that cannot pay all of it leaves the cards least overdue
		// standing: what the day before did not reach, and what falls due in
		// this one.
		slices.SortFunc(falls[today], func(a, b int) int { return older(cards[a], cards[b]) })
		spare = merges(spare[:0], due[take:], falls[today], cards)
		due, spare = spare, due
		take = 0
		clear(shown)
		again = again[:0]
		next := 0

		// What closed the day is every budget that turned a card away. A day
		// that asked for every card there was is closed by nothing.
		var closed BudgetNames
		if admits.Paused() {
			closed = closed.with(ClosedPaused)
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
		paid, settled, all := admits.Paused(), admits.Paused(), admits.Paused()
		for {
			owed := !paid && take < len(due)
			fresh := !all && left > 0
			back := !settled && next < len(again)
			if !owed && !fresh && !back {
				break
			}

			// A day hands over everything it owes and everything it begins
			// before it comes back to a card it has already shown, which is the
			// order the sittings of that day put them in.
			repeat := !owed && !fresh
			if repeat || admits.Paying(seen, begun, owed, fresh) {
				at := 0
				if repeat {
					at = again[next]
				} else {
					at = due[take]
				}
				counted := p.Counts.Charges(shown[at] > 0)
				if counted && admits.Closes.Reviews != ClosedNothing && charged >= admits.Reviews {
					closed = closed.with(admits.Closes.Reviews)
					if repeat {
						settled = true
					} else {
						paid = true
					}
					continue
				}
				if admits.Closes.Minutes != ClosedNothing && used+s.Cost.Review > admits.Minutes {
					closed = closed.with(admits.Closes.Minutes)
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
				cards[at] = s.answers(cards[at], open, ends, p, on)
				reckoned.answered(at, cards[at], ends)
				if s.Day.Owed(cards[at], open) && shown[at] < MostShowings {
					again = append(again, at)
					continue
				}
				if day := max(today+1, on.number(cards[at].Due)-base); day < days {
					falls[day] = append(falls[day], at)
				}
				continue
			}

			if admits.Closes.New != ClosedNothing && begun >= admits.New {
				closed, all = closed.with(admits.Closes.New), true
				continue
			}
			if admits.Closes.Minutes != ClosedNothing && used+s.Cost.New > admits.Minutes {
				closed, all = closed.with(admits.Closes.Minutes), true
				continue
			}
			used += s.Cost.New
			begun++
			answered++
			faced++
			left--
			out.Seen++
			one := s.answers(Schedule{}, open, ends, p, on)
			cards = append(cards, one)
			shown = append(shown, 1)
			reckoned.begun(one, ends)
			at := len(cards) - 1
			if s.Day.Owed(one, open) && shown[at] < MostShowings {
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
		out.Admitted = append(out.Admitted, !admits.Paused())
		out.Closed = append(out.Closed, closed)

		// How much of the material stands learned at the close of the day, which
		// is how far through it the day leaves a person, and how much of it
		// comes back at that hour.
		stands, back := reckoned.closes(cards, ends, out.Faces, answers[today])
		out.Through = append(out.Through, through(stands, out.Faces))
		if out.Learns == NeverLearns && stands == out.Faces {
			out.Learns = len(out.Load)
		}

		// What the day left standing is what fell due in it and was not reached,
		// and the day the backlog is gone is the first day none is.
		standing := len(due) - take + len(again) - next
		out.Backlog = append(out.Backlog, standing)
		if out.Clears == NeverClears && standing == 0 {
			out.Clears = len(out.Load)
		}
		if answers[today] {
			out.Retained.holds(today, back)
		}
		open = ends
	}

	if admitted := out.Admits(); admitted > 0 {
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

// merges puts what a day before did not reach and what falls due in this one
// into one run, the card face owed longest first.
//
// Both are in that order already, so a day sees the whole of what it owes for
// the cost of what it owes.
func merges(into, carried, fell []int, cards []Schedule) []int {
	a, b := 0, 0
	for a < len(carried) && b < len(fell) {
		if older(cards[carried[a]], cards[fell[b]]) <= 0 {
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
	preset  Preset
	carried bool
	target  float64
	learned []bool
	count   int
}

// reckons opens the count over the card faces a run begins with, at the instant
// it opens on.
func reckons(p Preset, cards []Schedule, at time.Time) *learnedCount {
	rule, _, retention := p.counting()
	out := &learnedCount{preset: p, carried: rule == RuleInterval, target: retention}
	if out.carried {
		out.learned = make([]bool, len(cards))
	}
	for i, c := range cards {
		if !p.Learned(c, at) {
			continue
		}
		out.count++
		if out.carried {
			out.learned[i] = true
		}
	}
	return out
}

// answered carries one card face the day has answered.
func (r *learnedCount) answered(card int, c Schedule, at time.Time) {
	if !r.carried {
		return
	}
	stands := r.preset.Learned(c, at)
	if stands == r.learned[card] {
		return
	}
	r.learned[card] = stands
	if stands {
		r.count++
		return
	}
	r.count--
}

// begun carries one card face the day has begun.
func (r *learnedCount) begun(c Schedule, at time.Time) {
	if !r.carried {
		return
	}
	stands := r.preset.Learned(c, at)
	r.learned = append(r.learned, stands)
	if stands {
		r.count++
	}
}

// closes is how many card faces stand learned at this instant, and what share of
// the material comes back at it where the day is one the run answers for. A card
// face nobody has begun comes back to nobody, and counts in the material.
//
// A carried count is a fact about a card face's schedule, so the walk over the
// whole material is made on the days the share is wanted. A count read off the
// chance of recall is the same walk, and the share falls out of it.
func (r *learnedCount) closes(
	cards []Schedule, at time.Time, faces int, wanted bool,
) (int, float64) {
	if r.carried {
		if !wanted || faces == 0 {
			return r.count, 0
		}
		back := 0.0
		for _, c := range cards {
			back += Recall(at.Sub(c.Last), c.Stability)
		}
		return r.count, back / float64(faces)
	}
	stands, back := 0, 0.0
	for _, c := range cards {
		one := Recall(at.Sub(c.Last), c.Stability)
		back += one
		if c.Seen() && one >= r.target {
			stands++
		}
	}
	if faces == 0 {
		return stands, 0
	}
	return stands, back / float64(faces)
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
	from := d.Opens(now)
	out := 0
	for range 7 {
		one := s.ripens(p, from)
		if one == NeverRipens {
			return NeverRipens
		}
		out = max(out, one)
		from = d.Ends(from)
	}
	return out
}

// ripens is how many days of review a card face begun on this day needs.
func (s Simulation) ripens(p Preset, open time.Time) int {
	var c Schedule
	days := 0
	for range LongestRipening {
		ends := s.Day.Ends(open)
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
		if c.Seen() && !c.Due.Before(s.Day.Ends(open)) {
			// Nothing is asked of it until the day its schedule falls in.
			open = s.Day.Opens(c.Due)
		}
		ends := s.Day.Ends(open)
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

// older puts the card face owed longest first. Two schedules alike in all of
// this are one card face as far as any of these numbers go.
func older(a, b Schedule) int {
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

// through is the share of the material learned. A preset scheduling nothing is
// through all of it.
func through(learned, faces int) float64 {
	if faces == 0 {
		return 1
	}
	return float64(learned) / float64(faces)
}
