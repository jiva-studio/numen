package flashcards

import (
	"cmp"
	"context"
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

// EvenFrom and EvenTo are the intervals a card may be moved within, in days.
// One falling short of the first or past the second stands where the scheduler
// put it.
const (
	EvenFrom = 2.5
	EvenTo   = Ahead
)

// Cost is how long an answer takes: one of a card being learned, and one of a
// card already learned.
type Cost struct{ New, Review time.Duration }

// DefaultCost is what a vault holding no answer times is projected at.
var DefaultCost = Cost{New: 20 * time.Second, Review: 8 * time.Second}

// Costed is how long an answer takes in this history, from the times the
// answers themselves carry. A kind of answer nobody has given yet stands at the
// default.
func Costed(by Scheduler, answers []Answer) Cost {
	var learning, learned time.Duration
	var learningCount, learnedCount int
	replayed(by, answers, func(before Schedule, a Answer) {
		took := min(a.Took, LongestAnswer)
		if took <= 0 {
			return
		}
		if by.Learned(before) {
			learned += took
			learnedCount++
			return
		}
		learning += took
		learningCount++
	})

	out := DefaultCost
	if learningCount > 0 {
		out.New = learning / time.Duration(learningCount)
	}
	if learnedCount > 0 {
		out.Review = learned / time.Duration(learnedCount)
	}
	return out
}

// CostedUnder is how long an answer takes under each preset, by the path the
// card faces are grouped under.
//
// A preset is costed from the answers to its own card faces: a preset of long
// cards and one of short cards turn the same minutes into different counts. A
// card face nothing groups is left out, and a kind of answer a preset holds
// none of stands at the default.
func CostedUnder(by Scheduler, answers []Answer, under map[CardFace]string) map[string]Cost {
	type taken struct {
		learning, learned           time.Duration
		learningCount, learnedCount int
	}
	held := make(map[string]*taken)
	replayed(by, answers, func(before Schedule, a Answer) {
		path, groups := under[a.CardFace]
		if !groups {
			return
		}
		took := min(a.Took, LongestAnswer)
		if took <= 0 {
			return
		}
		one := held[path]
		if one == nil {
			one = &taken{}
			held[path] = one
		}
		if by.Learned(before) {
			one.learned += took
			one.learnedCount++
			return
		}
		one.learning += took
		one.learningCount++
	})

	out := make(map[string]Cost, len(held))
	for path, one := range held {
		cost := DefaultCost
		if one.learningCount > 0 {
			cost.New = one.learning / time.Duration(one.learningCount)
		}
		if one.learnedCount > 0 {
			cost.Review = one.learned / time.Duration(one.learnedCount)
		}
		out[path] = cost
	}
	return out
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

// NewFSRSAt is the scheduler asking for this share of the cards to come back
// when they come round. A share outside what a preset may hold is brought to
// the nearest end of it.
func NewFSRSAt(retention float64) FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	p.RequestRetention = math.Min(math.Max(retention, RetentionBounds.Least), RetentionBounds.Most)
	return FSRS{engine: fsrs.NewFSRS(p), name: FSRSName + "." + weighed(p)}
}

// Projection is what a preset comes to over the days ahead.
//
// Everything in it is a number a caller shows. It is worked out from the
// schedules the answers have already produced, and nothing in it is written
// anywhere.
type Projection struct {
	// Days is how many days were projected.
	Days int
	// ReviewsADay and MinutesADay are the daily load, over the days projected.
	ReviewsADay float64
	MinutesADay float64
	// Retained is the share of the material that comes back on the last day.
	Retained float64
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
	// Load is how many answers each day projected carried, and Spent is how
	// long those answers took. The first of each is the day a sitting now would
	// ask, which is the day a caller shows against a control.
	Load  []int
	Spent []time.Duration
	// Closed is what stopped each day projected asking for more.
	Closed []Closed
	// Backlog is how many card faces stood overdue at the end of each day
	// projected: their day had passed and that day did not get to them. It is
	// the pile a person watches shrink, and it begins where Overdue stands now.
	Backlog []int
	// Through is the share of the material answered at least once by the end of
	// each day projected.
	Through []float64
}

// NeverClears is a pace that leaves something overdue on every day projected.
const NeverClears = -1

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

// Closed is what stopped a day of review asking for more.
//
// A day that asked for everything there was is closed by nothing: the material
// ran out. The three others name the budget in the words the preset writes it
// in, and a budget the goal does not name can never be one of them.
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
}

// Run projects the card faces forward from now.
//
// The map is where the answers have left every card face the preset schedules,
// and unseen is how many of its card faces nobody has answered. A day's answers
// are all given at the hour the day opens, and a card face is answered at most
// once in a day.
//
// A run over many days is long enough that a caller may give up on it, so the
// day it is on is where it is left.
func (s Simulation) Run(
	ctx context.Context, now time.Time, p Preset, at map[CardFace]Schedule, unseen int,
) (Projection, error) {
	days := s.Days
	if days <= 0 {
		days = Ahead
	}

	// The map hands its schedules over in whatever order it holds them, and a
	// projection asked twice answers the same both times.
	cards := make([]Schedule, 0, len(at))
	for _, one := range at {
		cards = append(cards, one)
	}
	slices.SortFunc(cards, older)

	out := Projection{
		Days: days, Faces: len(at) + unseen, Seen: len(at), Clears: NeverClears,
	}
	// A day that begins with nothing overdue has nothing to clear.
	if Overdue(s.Day, at, now) == 0 {
		out.Clears = 0
	}
	left := unseen
	var spent time.Duration
	// Which day of the projection last answered each card face, so a card the
	// day has just had is not counted as one the day left standing.
	answeredOn := make([]int, len(cards), len(cards)+unseen)
	for i := range answeredOn {
		answeredOn[i] = -1
	}

	open := s.Day.Ends(now).AddDate(0, 0, -1)
	// Where the answers so far have left every card face is what the days
	// ahead are loaded with.
	on := Spreading(s.Day)
	for _, c := range cards {
		on.Holds(c.Due)
	}
	for today := range days {
		if err := ctx.Err(); err != nil {
			return Projection{}, err
		}
		ends := s.Day.Ends(open)
		var used time.Duration

		// What the day admits is the one answer, and it is the answer the
		// sitting of that day will be held to.
		admits := p.Admits(s.Day, open, Spent{}, left)

		var due []int
		for i, c := range cards {
			if c.Due.Before(ends) {
				due = append(due, i)
			}
		}
		// The oldest debt is paid first, so a day that cannot pay all of it
		// leaves the cards least overdue standing.
		slices.SortFunc(due, func(a, b int) int { return older(cards[a], cards[b]) })

		// What closed the day is the budget that turned a card away. A day that
		// asked for every card there was is closed by nothing.
		closed := ClosedNothing
		if admits.Paused {
			closed = ClosedPaused
		}

		// The day is spent between the debt and the material it has not begun,
		// in the share the preset names. A side the day has no more room for is
		// done with, and the other goes on with what is left of the day.
		answered, seen, begun, take := 0, 0, 0, 0
		paid, all := admits.Paused, admits.Paused
		for !paid || !all {
			owed, fresh := !paid && take < len(due), !all && left > 0
			if !owed {
				paid = true
			}
			if !fresh {
				all = true
			}
			if !owed && !fresh {
				break
			}

			if admits.Paying(seen, begun, owed, fresh) {
				if admits.Closes.Reviews != ClosedNothing && seen >= admits.Reviews {
					closed, paid = admits.Closes.Reviews, true
					continue
				}
				if admits.Closes.Minutes != ClosedNothing && used+s.Cost.Review > admits.Minutes {
					closed, paid = admits.Closes.Minutes, true
					continue
				}
				at := due[take]
				take++
				used += s.Cost.Review
				seen++
				answered++
				cards[at] = s.step(cards[at], open, p, on)
				answeredOn[at] = today
				continue
			}

			if admits.Closes.New != ClosedNothing && begun >= admits.New {
				closed, all = admits.Closes.New, true
				continue
			}
			if admits.Closes.Minutes != ClosedNothing && used+s.Cost.New > admits.Minutes {
				closed, all = admits.Closes.Minutes, true
				continue
			}
			used += s.Cost.New
			begun++
			answered++
			left--
			out.Seen++
			cards = append(cards, s.step(Schedule{}, open, p, on))
			answeredOn = append(answeredOn, today)
		}

		out.Answered += answered
		spent += used
		out.Load = append(out.Load, answered)
		out.Spent = append(out.Spent, used)
		out.Closed = append(out.Closed, closed)
		out.Through = append(out.Through, through(out.Seen, out.Faces))

		// What the day left standing, and the day the backlog is gone.
		standing := behind(cards, answeredOn, ends, today)
		out.Backlog = append(out.Backlog, standing)
		if out.Clears == NeverClears && standing == 0 {
			out.Clears = len(out.Load)
		}
		open = ends
	}

	out.ReviewsADay = float64(out.Answered) / float64(days)
	out.MinutesADay = spent.Minutes() / float64(days)
	for _, c := range cards {
		if c.Due.Before(open) {
			out.Owed++
		}
		out.Retained += Recall(open.Sub(c.Last), c.Stability)
	}
	if out.Faces > 0 {
		out.Retained /= float64(out.Faces)
	}
	return out, nil
}

// behind is how many card faces this day left standing: their day has passed
// and the day did not get to them.
//
// A card the day answered is not one of them, whatever the scheduler did with
// it: a card begun this morning and asked for again ten minutes later has had
// its day, and is picked up in the next.
func behind(cards []Schedule, answeredOn []int, at time.Time, today int) int {
	out := 0
	for i, c := range cards {
		if c.Due.Before(at) && answeredOn[i] != today {
			out++
		}
	}
	return out
}

// step is where one projected answer leaves a card face.
//
// Both endings are worked out and weighed by how likely the card is to come
// back, so a projection follows one card down the middle of what it may do. The
// phase is the one a card that came back is left in.
func (s Simulation) step(c Schedule, at time.Time, p Preset, on *Spread) Schedule {
	good := s.By.Next(c, at, Good)
	if !c.Seen() {
		good.Due = p.Places(on, at, good.Due)
		return good
	}
	back := Recall(at.Sub(c.Last), c.Stability)
	again := s.By.Next(c, at, Again)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(at).Seconds() + (1-back)*again.Due.Sub(at).Seconds()
	out.Due = p.Places(on, at, at.Add(time.Duration(away*float64(time.Second))))
	return out
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
// It is the one place a day is chosen. A sitting and a projection of it both
// come here.
func (p Preset) Places(s *Spread, at, due time.Time) time.Time {
	if s == nil {
		return due
	}
	first, last, opens := window(due.Sub(at))
	if !p.EvenLoad || !opens {
		s.Holds(due)
		return due
	}

	stands := s.number(due)
	from, to := s.number(at.AddDate(0, 0, first)), s.number(at.AddDate(0, 0, last))
	on, heaviest := stands, -1.0
	if stands >= from && stands <= to {
		heaviest = p.weighs(s, stands)
	}
	for day := from; day <= to; day++ {
		if weight := p.weighs(s, day); weight > heaviest {
			on, heaviest = day, weight
		}
	}

	out := due.AddDate(0, 0, on-stands)
	s.Holds(out)
	return out
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

// through is the share of the material answered at least once. A preset
// scheduling nothing is through all of it.
func through(seen, faces int) float64 {
	if faces == 0 {
		return 1
	}
	return float64(seen) / float64(faces)
}
