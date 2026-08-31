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

// LightShare is how much of a day's load a day named light carries.
const LightShare = 0.5

// EvenSlack is how far an even load moves a review, as a share of the interval
// it stands at. A card moved this far comes round when a person expects it.
const EvenSlack = 0.05

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
// Light days and an even load move cards between neighbouring days and leave
// the load over a week where it was.
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
	var even *spread
	if p.EvenLoad {
		even = &spread{from: open, on: make(map[int]int, len(cards))}
		for _, c := range cards {
			even.on[even.day(c.Due)]++
		}
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

		answered, seen := 0, 0
		for _, i := range due {
			if admits.Paused {
				break
			}
			if admits.Closes.Reviews != ClosedNothing && seen >= admits.Reviews {
				closed = admits.Closes.Reviews
				break
			}
			if admits.Closes.Minutes != ClosedNothing && used+s.Cost.Review > admits.Minutes {
				closed = admits.Closes.Minutes
				break
			}
			used += s.Cost.Review
			seen++
			answered++
			cards[i] = s.step(cards[i], open, even)
			answeredOn[i] = today
		}

		for begun := 0; left > 0 && !admits.Paused; begun++ {
			if admits.Closes.New != ClosedNothing && begun >= admits.New {
				closed = admits.Closes.New
				break
			}
			if admits.Closes.Minutes != ClosedNothing && used+s.Cost.New > admits.Minutes {
				closed = admits.Closes.Minutes
				break
			}
			used += s.Cost.New
			answered++
			left--
			out.Seen++
			cards = append(cards, s.step(Schedule{}, open, even))
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
func (s Simulation) step(c Schedule, at time.Time, even *spread) Schedule {
	good := s.By.Next(c, at, Good)
	if !c.Seen() {
		good.Due = even.place(at, good.Due)
		return good
	}
	back := Recall(at.Sub(c.Last), c.Stability)
	again := s.By.Next(c, at, Again)

	out := good
	out.Stability = back*good.Stability + (1-back)*again.Stability
	out.Difficulty = back*good.Difficulty + (1-back)*again.Difficulty
	away := back*good.Due.Sub(at).Seconds() + (1-back)*again.Due.Sub(at).Seconds()
	out.Due = even.place(at, at.Add(time.Duration(away*float64(time.Second))))
	return out
}

// Budget is what one day of a preset holds: how many cards of each kind, and
// how long the day runs.
type Budget struct {
	New     int
	Reviews int
	Minutes float64
}

// on is the budget a preset keeps on this day of the week. A day named light
// carries LightShare of the load, and what it sheds stands on its neighbours.
//
// What a day of it admits is Admits, which is the one place a limit is read.
func (p Preset) on(day time.Weekday) Budget {
	share := weekly(p.LightDays)[day]
	return Budget{
		New:     int(math.Round(share * float64(p.NewADay))),
		Reviews: int(math.Round(share * float64(p.ReviewsADay))),
		Minutes: share * float64(p.MinutesADay),
	}
}

// weekly is how much of a day's load each day of the week carries.
//
// A day named light carries LightShare of it and sheds the rest to the nearest
// day either side that is not light, so a week carries what it did. A week of
// nothing but light days is a week of ordinary ones.
func weekly(light []time.Weekday) [7]float64 {
	var out [7]float64
	var cut [7]bool
	named := 0
	for _, day := range light {
		if day >= 0 && int(day) < len(cut) && !cut[day] {
			cut[day] = true
			named++
		}
	}
	for i := range out {
		out[i] = 1
	}
	if named == 0 || named == len(cut) {
		return out
	}
	for day := range out {
		if !cut[day] {
			continue
		}
		out[day] = LightShare
		shed := (1 - LightShare) / 2
		out[toward(cut, day, -1)] += shed
		out[toward(cut, day, 1)] += shed
	}
	return out
}

// toward is the nearest day of the week that is not light, walked round this
// way.
func toward(cut [7]bool, from, step int) int {
	at := from
	for {
		at = (at + step + len(cut)) % len(cut)
		if !cut[at] {
			return at
		}
	}
}

// spread is where an even load puts a card whose next review may fall on any of
// a few days: the day of them carrying least.
type spread struct {
	from time.Time
	on   map[int]int
}

// day is which day of the projection an instant falls in.
func (s *spread) day(at time.Time) int {
	return int(math.Floor(at.Sub(s.from).Hours() / 24))
}

// place is the day a review is put on, and counts the card against it. A
// projection keeping no even load leaves the review where the scheduler put it.
func (s *spread) place(at, due time.Time) time.Time {
	if s == nil {
		return due
	}
	away := due.Sub(at)
	if away <= 0 {
		return due
	}
	slack := time.Duration(EvenSlack * float64(away))
	stands := s.day(due)
	first, last := max(s.day(due.Add(-slack)), s.day(at)+1), s.day(due.Add(slack))

	on := stands
	for day := first; day <= last; day++ {
		// A day tied with the one the scheduler named leaves the card where it is.
		if s.on[day] < s.on[on] {
			on = day
		}
	}
	s.on[on]++
	return due.AddDate(0, 0, on-stands)
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
