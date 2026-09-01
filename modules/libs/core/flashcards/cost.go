package flashcards

import (
	"cmp"
	"context"
	"maps"
	"math"
	"slices"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// Ahead is how many days a projection runs when it is not told.
const Ahead = 90

// LongestAnswer is the most one answer is counted at. A card left on the screen
// while a person answered the door stands there for an hour, and the hour is
// not review.
const LongestAnswer = time.Minute

// LeastStability is the least a projected card face is taken to stand at, in
// days, which is a minute. The scheduler divides by stability, so a card face a
// long run of lapses has worn down to none of it answers with a number nobody
// can read, and every figure weighed against it after that is that number.
const LeastStability = 1.0 / (24 * 60)

// EvenFrom and EvenTo are the intervals a card may be moved within, in days.
// One falling short of the first or past the second stands where the scheduler
// put it.
const (
	EvenFrom = 2.5
	EvenTo   = Ahead
)

// ShortestAnswer is the shortest a kind of answer is costed at. A card graded
// before it could be read is a key hit and not review.
const ShortestAnswer = time.Second

// MostShowings is how many times one day of review asks a card face. An answer
// a card did not come back on sends it away for minutes, and the day it lands
// back in is the day it was asked in; a card that keeps landing there is put
// down and picked up by the day after.
const MostShowings = 8

// LeastAnswers is how many answers of a kind a history holds before it says
// what that kind costs. A kind the history holds fewer of stands at the
// default.
const LeastAnswers = 10

// Cost is how long an answer takes: one of a card the scheduler is still
// putting into memory, and one of a card that comes round in days.
//
// ReadNew and ReadReview say which halves the history answered. A half it does
// not answer stands at the default, and a caller putting the number in front of
// a person says which of the two it is showing.
type Cost struct {
	New, Review         time.Duration
	ReadNew, ReadReview bool
}

// DefaultCost is what a vault holding no answer times is projected at.
var DefaultCost = Cost{New: 20 * time.Second, Review: 8 * time.Second}

// Costed is how long an answer takes in this history, from the times the
// answers themselves carry. A kind of answer the history holds too few of
// stands at the default.
func Costed(by Scheduler, answers []Answer) Cost {
	var took taking
	replayed(by, answers, func(before Schedule, a Answer) {
		took.holds(by.Spaced(before), a.Took)
	})
	return took.cost()
}

// CostedUnder is how long an answer takes under each preset, by the path the
// card faces are grouped under.
//
// A preset is costed from the answers to its own card faces: a preset of long
// cards and one of short cards turn the same minutes into different counts. A
// card face nothing groups is left out, and a kind of answer a preset holds too
// few of stands at the default.
func CostedUnder(by Scheduler, answers []Answer, under map[CardFace]string) map[string]Cost {
	held := make(map[string]*taking)
	replayed(by, answers, func(before Schedule, a Answer) {
		path, groups := under[a.CardFace]
		if !groups {
			return
		}
		one := held[path]
		if one == nil {
			one = &taking{}
			held[path] = one
		}
		one.holds(by.Spaced(before), a.Took)
	})

	out := make(map[string]Cost, len(held))
	for path, one := range held {
		out[path] = one.cost()
	}
	return out
}

// taking is how long the answers of each kind took, one entry an answer.
type taking struct{ begun, spaced []time.Duration }

// holds counts one answer, capped at LongestAnswer. An answer carrying no time
// at all says nothing about how long its kind takes.
func (t *taking) holds(spaced bool, took time.Duration) {
	took = min(took, LongestAnswer)
	if took <= 0 {
		return
	}
	if spaced {
		t.spaced = append(t.spaced, took)
		return
	}
	t.begun = append(t.begun, took)
}

// cost is what these answers say a kind of answer takes, each half standing at
// the default where the history is too short to say.
func (t *taking) cost() Cost {
	out := DefaultCost
	if middle, read := middling(t.begun); read {
		out.New, out.ReadNew = middle, true
	}
	if middle, read := middling(t.spaced); read {
		out.Review, out.ReadReview = middle, true
	}
	return out
}

// middling is the middle of these answers, and whether there are enough of them
// to have one.
//
// Answer times are a right tail: a person answers the door, and the card stands
// on the screen while they do. The middle is where half the answers fall either
// side of it, and one long answer moves it by one place.
func middling(took []time.Duration) (time.Duration, bool) {
	if len(took) < LeastAnswers {
		return 0, false
	}
	slices.Sort(took)
	out := took[len(took)/2]
	if len(took)%2 == 0 {
		out = (took[len(took)/2-1] + out) / 2
	}
	return max(out, ShortestAnswer), true
}

// The forgetting curve a projection reads a stability by.
var (
	recallFactor = fsrs.DefaultParam().Factor
	recallDecay  = fsrs.DefaultParam().Decay
)

// Recall is the share of cards standing at this stability that come back after
// this long away. A card nothing is known about comes back to nobody.
func Recall(away time.Duration, stability float64) float64 {
	if stability <= 0 {
		return 0
	}
	days := math.Max(away.Hours()/24, 0)
	return math.Pow(1+recallFactor*days/stability, recallDecay)
}

// Recalling is what a projection assumes about coming back: how likely a card
// face standing here is to be recalled when it is asked at this instant.
//
// A projection follows one card down the middle of what it may do, weighing the
// ending where it came back against the ending where it did not, and this is
// the weight. Every figure a run draws is drawn under the assumption it was
// given, so a caller naming one says which.
type Recalling func(Schedule, time.Time) float64

// AsModelled is the chance the scheduler's own forgetting curve gives a card
// face, and is what a run not told otherwise reads.
func AsModelled(c Schedule, at time.Time) float64 {
	return Recall(at.Sub(c.Last), c.Stability)
}

// NothingForgotten is a run in which every card face asked comes back.
func NothingForgotten(Schedule, time.Time) float64 { return 1 }

// NewFSRSAt is the scheduler asking for this share of the cards to come back
// when they come round. A share outside what a preset may hold is brought to
// the nearest end of it.
func NewFSRSAt(retention float64) FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	p.RequestRetention = math.Min(math.Max(retention, RetentionBounds.Least), RetentionBounds.Most)
	return FSRS{p: p, name: FSRSName + "." + weighed(p)}
}

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
	Retained Kept
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
	Closed []Closing
	// Backlog is how many card faces stood overdue at the end of each day
	// projected: their day had passed and that day did not get to them. It is
	// the pile a person watches shrink, and it begins where Overdue stands now.
	Backlog []int
	// Through is the share of the material learned by the end of each day
	// projected, under the rule the preset names. Getting through the material
	// is learning it, and there is no second reckoning of it.
	Through []float64
}

// Kept is the share of the material that comes back at the end of a day, on the
// days a run was asked to answer for. A day it was not asked for holds no share,
// and On says so.
type Kept struct{ on map[int]float64 }

// On is the share of the material that came back at the end of this day of the
// run, counting the day the run opens as none, and whether the run answers for
// that day.
func (k Kept) On(day int) (float64, bool) {
	share, answers := k.on[day]
	return share, answers
}

// Days is every day this answers for, in order.
func (k Kept) Days() []int { return slices.Sorted(maps.Keys(k.on)) }

// holds the share one day came to.
func (k *Kept) holds(day int, share float64) {
	if k.on == nil {
		k.on = make(map[int]float64)
	}
	k.on[day] = share
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
func Overdue(d Day, at map[CardFace]Schedule, now time.Time) int {
	opened := d.Ends(now).AddDate(0, 0, -1)
	out := 0
	for _, s := range at {
		if s.Seen() && s.Due.Before(opened) {
			out++
		}
	}
	return out
}

// Closed is one budget a day of review may be stopped by, in the words the
// preset writes the key in. A budget the goal does not name is never one of
// them, and a day that asked for everything there was is closed by nothing:
// the material ran out.
type Closed string

const (
	ClosedNothing Closed = ""
	ClosedMinutes Closed = "minutes_a_day"
	ClosedNew     Closed = "new_a_day"
	ClosedReviews Closed = "reviews_a_day"
	// ClosedDate is a day paced by the day the preset aims at.
	ClosedDate Closed = "by_date"
	// ClosedBacklog is the share of a day that goes to the debt. It closes
	// nothing, and stands in this vocabulary because it is a key of a preset
	// that a goal either reads or leaves idle.
	ClosedBacklog Closed = "backlog"
	// ClosedPaused is a preset scheduling nothing at all.
	ClosedPaused Closed = "paused"
)

// Closing is every budget that closed one day of review.
//
// A goal of retention holds a day to both card counts, and a day that ran out
// of new cards and of reviews names both: a person raising one of them and
// finding nothing changed is reading a day the other closed too.
type Closing []Closed

// closers is every budget a day may be closed by, in the order a closing names
// them.
var closers = []Closed{ClosedPaused, ClosedMinutes, ClosedNew, ClosedReviews, ClosedDate}

// Holds reports whether this budget is one of those that closed the day.
func (c Closing) Holds(one Closed) bool { return slices.Contains(c, one) }

// with is this closing and one budget more, named once and in the order closers
// stands in.
func (c Closing) with(one Closed) Closing {
	if !slices.Contains(closers, one) || c.Holds(one) {
		return c
	}
	out := make(Closing, 0, len(c)+1)
	for _, each := range closers {
		if each == one || c.Holds(each) {
			out = append(out, each)
		}
	}
	return out
}

// Names is every budget that closed the day, each in the words the preset
// writes the key in.
func (c Closing) Names() []string {
	out := make([]string, 0, len(c))
	for _, one := range c {
		out = append(out, string(one))
	}
	return out
}

// Name is the key one budget is written under, and empty where the day was
// closed by none or by more than one.
func (c Closing) Name() string {
	if len(c) != 1 {
		return ""
	}
	return string(c[0])
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
	Cost Cost
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
	Recalls Recalling
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
	ctx context.Context, now time.Time, p Preset, at map[CardFace]Schedule, unseen int,
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

	open := s.Day.Ends(now).AddDate(0, 0, -1)
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
		admits := p.Admits(s.Day, open, gone, Left{New: left, Ripens: ripens})

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
		var closed Closing
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

// reckoning is how much of the material stands learned and how much of it comes
// back, at the close of a day.
//
// Under a rule of an interval what stands learned is a fact about a card face's
// schedule: the count is carried from day to day and asked again only of a card
// face the day answered. Under a rule of a chance of recall it is a fact about
// the instant, and is read off the same number as the share that comes back.
type reckoning struct {
	preset  Preset
	carried bool
	target  float64
	learned []bool
	count   int
}

// reckons opens the count over the card faces a run begins with, at the instant
// it opens on.
func reckons(p Preset, cards []Schedule, at time.Time) *reckoning {
	rule, _, retention := p.counting()
	out := &reckoning{preset: p, carried: rule == RuleInterval, target: retention}
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
func (r *reckoning) answered(card int, c Schedule, at time.Time) {
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
func (r *reckoning) begun(c Schedule, at time.Time) {
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
func (r *reckoning) closes(
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
	from := d.Ends(now).AddDate(0, 0, -1)
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
func (s Simulation) answers(c Schedule, open, ends time.Time, p Preset, on *Spread) Schedule {
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
			open = s.Day.Ends(c.Due).AddDate(0, 0, -1)
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
func (s Simulation) step(c Schedule, at time.Time, p Preset, on *Spread) Schedule {
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

// Budget is what one day of a preset holds: how many cards of each kind, and
// how long the day runs.
type Budget struct {
	New     int
	Reviews int
	Minutes float64
}

// on is the budget a preset keeps on this day of the week: its share of the
// load, whether or not the days are evened out.
//
// What a day of it admits is Admits, which is the one place a limit is read.
func (p Preset) on(day time.Weekday) Budget {
	share := p.Share(day)
	return Budget{
		New:     int(math.Round(share * float64(p.NewADay))),
		Reviews: int(math.Round(share * float64(p.ReviewsADay))),
		Minutes: share * float64(p.MinutesADay),
	}
}

// Spread is how loaded each day of review is: how many card faces fall on each.
//
// It is one table over every preset, and the answers replayed and the
// projection ahead of them both read it.
type Spread struct {
	day Day
	on  map[int]int
}

// Spreading opens a table counting the days as this day of review divides them.
func Spreading(d Day) *Spread { return &Spread{day: d, on: make(map[int]int)} }

// Holds counts one card face against the day its schedule falls in.
func (s *Spread) Holds(due time.Time) {
	if s != nil {
		s.on[s.number(due)]++
	}
}

// On is how many card faces fall on the day of review holding this instant.
func (s *Spread) On(at time.Time) int {
	if s == nil {
		return 0
	}
	return s.on[s.number(at)]
}

// number is the day of review holding an instant, as a whole number counted
// from the day the clock is counted from. The same answers name the same days
// in every process.
func (s *Spread) number(at time.Time) int {
	opened := s.day.Ends(at).AddDate(0, 0, -1)
	y, m, d := opened.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / int64(24*time.Hour/time.Second))
}

// weekday is the day of the week a numbered day of review falls on. The day the
// clock is counted from was a Thursday.
func weekday(number int) time.Weekday {
	return time.Weekday(((number+int(time.Thursday))%7 + 7) % 7)
}

// Places is the day a card answered at this instant comes back on, and counts
// it against that day.
//
// It is chosen inside the tolerance the scheduler allows around the interval it
// worked out. Every day of that window carries a weight — the share of the load
// its day of the week keeps, over what already falls on it — and the heaviest
// takes the card, the day the scheduler named winning a tie. A day at nothing
// weighs nothing and takes no card.
//
// It is pressure and not a promise: no day is forbidden to carry more than its
// share, and a preset keeping no even load leaves the card where it fell.
//
// A preset aiming at a day moves no card face. The days beyond the front of a
// projection carry nothing, so the lightest day of a window is its last, and a
// card put there is a card asked for later. The pace is what spreads a date's
// material over its days, and the days it has are the days it needs.
//
// It is the one place a day is chosen. A sitting and a projection of it both
// come here.
func (p Preset) Places(s *Spread, at, due time.Time) time.Time {
	out := p.lands(s, at, due)
	s.Holds(out)
	return out
}

// Lands is the day a card answered at this instant would come back on, counting
// it against no day.
//
// The four windows put to a person are four askings of one card, and one of
// them is answered. The day each of them names is chosen by Places' own
// arithmetic, so the button names the day the card lands on.
func (p Preset) Lands(s *Spread, at, due time.Time) time.Time {
	return p.lands(s, at, due)
}

// lands is where the day is chosen.
func (p Preset) lands(s *Spread, at, due time.Time) time.Time {
	if s == nil {
		return due
	}
	first, last, opens := window(due.Sub(at))
	if !p.Evens() || !opens {
		return due
	}

	// A day is added on the clock the boundary between days is read from. The
	// moment a card lands on is the same whichever zone the instant arrives in.
	in := s.day.zone()
	counted := at.In(in)

	stands := s.number(due)
	from, to := s.number(counted.AddDate(0, 0, first)), s.number(counted.AddDate(0, 0, last))
	on, heaviest := stands, -1.0
	if stands >= from && stands <= to {
		heaviest = p.weighs(s, stands)
	}
	for day := from; day <= to; day++ {
		if weight := p.weighs(s, day); weight > heaviest {
			on, heaviest = day, weight
		}
	}

	// The instant is handed back in the zone it arrived in.
	return due.In(in).AddDate(0, 0, on-stands).In(due.Location())
}

// weighs is how much a numbered day of review wants another card: the share of
// the load its day of the week keeps, over what already falls on it.
func (p Preset) weighs(s *Spread, day int) float64 {
	return p.Share(weekday(day)) / float64(1+s.on[day])
}

// slacks is how far either side of an interval a card may be put, by how long
// the interval is: the wider it is, the more days come round when a person
// expects them.
var slacks = []struct{ from, to, factor float64 }{
	{2.5, 7, 0.15},
	{7, 20, 0.1},
	{20, math.Inf(1), 0.05},
}

// window is the days either side of an interval a card may be put on, counted
// from the answer, and whether the window holds more than one day.
func window(away time.Duration) (first, last int, opens bool) {
	days := away.Hours() / 24
	if days < EvenFrom || days > EvenTo {
		return 0, 0, false
	}
	slack := 1.0
	for _, one := range slacks {
		slack += one.factor * math.Max(math.Min(days, one.to)-one.from, 0)
	}
	first = int(math.Max(2, math.Round(days-slack)))
	last = int(math.Round(days + slack))
	return first, last, last > first
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
