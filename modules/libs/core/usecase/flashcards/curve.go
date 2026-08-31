package flashcards

import (
	"context"
	"math"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// Points is how many places a curve is worked out at. The whole range is
// worked out in one pass, so a control moving over it reads a finished array.
const Points = 25

// LeastCeiling is the shortest day a curve of minutes runs to.
const LeastCeiling = 60

// MostAhead is how far ahead a goal of a date is projected. A day further off
// than this gets no curve.
const MostAhead = 5 * 365

// Curve is what the one control of a preset comes to over the whole range of
// its goal.
//
// Nothing in it is worked out again as the control moves: the grid, what stands
// at each place of it, and the two marks are all here.
type Curve struct {
	Goal history.Goal
	// Grid is the value of the goal at each place: minutes for minutes_a_day, a
	// share of cards for retention, and days from today for by_date.
	Grid []float64
	// Days names the day of each place, and is filled for a goal of a date.
	Days []string
	// At is what the preset comes to at each place of the grid.
	At []Point
	// Now is where the preset stands.
	Now Mark
	// Suggested is the place worth pointing at: under a goal of minutes the
	// shortest day the clock no longer cuts short, and under a goal of a date
	// the first day the budget the preset keeps gets through the material.
	//
	// A goal of retention has none, and stands at Nowhere. What a target leaves
	// in the head climbs the whole way to the top of the range, so a mark on the
	// most of it would sit at the far end of every curve and advise only asking
	// for as much as memory allows.
	Suggested Mark
	// Decks is how many decks are scheduled by this preset. Zero is a preset no
	// deck points at, and every place of the curve stands at zero with it.
	Decks int
	// Cards is how many card faces stand in those decks. Zero is a preset with
	// nothing to schedule, and every place of the curve stands at zero with it.
	Cards int
	// Overdue is how many of those card faces have had their day and were not
	// answered on it, and Unbegun how many nobody has answered at all, so they
	// have had no day. Both are one number over the whole curve: facts about the
	// vault as it stands, and not about the setting being chosen.
	Overdue int
	Unbegun int
}

// Point is what a preset comes to at one place of the grid.
//
// What Reviews and Minutes are the height of is the goal's own question.
//
// Under a goal of minutes and a goal of a date they are the first day of the
// run the preset admits: the next sitting a person will actually sit down to. A
// day at none of the load is no sitting at all, so the picture draws the day
// after it, and a person moving a control on such a day reads what the setting
// buys them rather than a row of noughts.
//
// It is one real day of the run, worked out by the arithmetic the deck screen
// runs, so the count read off the curve is the count that sitting hands a
// person.
//
// Under a goal of retention they are the load over the days the preset admits.
// A target does nothing to today and everything to the weeks after it, so every
// place of that grid would otherwise draw the same day.
//
// A goal of a date fills Minutes with what getting through the material by that
// day costs and Owed with the backlog that pace leaves standing on it, and
// Through and Enough with what the budget the preset keeps gets through by it.
type Point struct {
	Reviews float64
	Minutes float64
	// Retained is the share of the material that comes back on the last day
	// projected, which is what the load beside it buys.
	Retained float64
	// Owed is the card faces standing owed on the last day the projection ran:
	// Ahead days off under a goal of minutes or of retention, and the day the
	// place names under a goal of a date. It is the backlog left at the end and
	// not the debt a day carries.
	Owed int
	// Through is the share of the material learned by this day, and Enough and
	// Met are whether the pace this place sets learns every card face that can
	// be learned by it. Under a date the pace is the budget, so the two are one
	// question and are read off the one run the place is drawn from.
	Through float64
	Enough  bool
	Met     bool
	// Short is how many card faces cannot be learned by this day whatever the
	// pace, which is the rule wanting more days than the day leaves them.
	Short int
	// Closed is the budget that closed the day here, in the words the preset
	// writes it in, and is empty where the material itself ran out.
	Closed history.Closed
	// Clears is how many days of review at this place it takes before nothing
	// is overdue. A curve standing over nothing overdue clears in none, and a
	// place whose pace never gets there is history.NeverClears.
	Clears int
	// Learned is how many card faces stand learned today at this place, and
	// Learns how many days of review it takes before all of them do. A place
	// whose horizon ends with one still to learn is history.NeverLearns, and one
	// with no such day to name is history.LearnsUnasked.
	Learned int
	Learns  int
	// Backlog is how many card faces stand overdue at the end of each day
	// projected at this place, one entry a day over the whole horizon. It runs
	// over days and not over the goal's range, so it is drawn beside the curve
	// rather than under it.
	Backlog []int
}

// Mark is one place on the curve worth pointing at.
type Mark struct {
	// At is the place of the grid, and is -1 when the value falls outside it.
	At int
	// Value is the goal's value at the mark, in the units of the grid. What the
	// preset stands at need not sit on the grid.
	Value float64
	// Day is the day at the mark, and is filled for a goal of a date.
	Day string
}

// Nowhere is a mark that falls outside the grid.
var Nowhere = Mark{At: -1}

// Curves is the simulator behind the one control of a preset.
//
// It reads the vault's answers once and projects them forward at every place of
// the goal's range. Nothing here writes: a curve is shown beside a control, and
// what the control settles is written by the person moving it.
type Curves struct {
	Standings Standings
	Schedules Schedules
	// Presets says which preset each deck is scheduled by. A build holding no
	// links projects every deck of the vault.
	Presets Presets
	Day     history.Day
	Now     func() time.Time
	// At is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	At func(retention float64) history.Scheduler
}

// Execute is the curve of the goal this preset steers.
//
// The preset is passed in: what a curve is wanted for is the value a person is
// moving and has not written yet. Path is the note the preset stands in and
// names the decks it schedules, and a preset standing in no note schedules the
// decks that name none.
func (u Curves) Execute(
	ctx context.Context, v domain.Vault, path string, p history.Preset,
) (Curve, error) {
	standing, err := u.Standings.Execute(ctx, v)
	if err != nil {
		return Curve{}, err
	}
	held, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return Curve{}, err
	}
	// A deck is asked once which preset schedules it, however many card faces
	// it holds, and a preset note is opened once however many decks name it.
	reading := u.Presets.Reading()
	asks, err := u.Schedules.under(ctx, v, reading, standing)
	if err != nil {
		return Curve{}, err
	}
	schedules := u.Schedules.worked(ctx, v, held, asks)

	decks := make(map[string]bool)
	at := make(map[history.CardFace]history.Schedule)
	// The card faces this preset schedules, which is what it is costed from.
	under := make(map[history.CardFace]string)
	unseen := 0
	for _, one := range standing {
		mine, asked := decks[one.Deck]
		if !asked {
			held, err := reading.Of(ctx, v, one.Deck)
			if err != nil {
				return Curve{}, err
			}
			mine = held.Path == path
			decks[one.Deck] = mine
		}
		if !mine {
			continue
		}
		under[one.CardFace] = path
		s, answered := schedules[one.CardFace]
		if !answered {
			unseen++
			continue
		}
		at[one.CardFace] = s
	}

	// How many decks this preset schedules, counted over every deck the vault
	// holds: a deck of no cards points at its preset like any other.
	mine, err := u.pointing(ctx, v, reading, path, decks)
	if err != nil {
		return Curve{}, err
	}

	cost, costed := history.CostedUnder(u.Schedules.By, held.Answers, under)[path]
	if !costed {
		cost = history.DefaultCost
	}
	now := u.now()
	// The projection is run by the scheduler this preset asks for, which is the
	// one its cards are scheduled by, and it opens on the day a person is
	// already partway through.
	run := history.Simulation{
		By: u.at(p.Retention), Day: u.Day, Cost: cost,
		Spent: history.Sat(u.Day, u.Day.Names(now), held.Answers, under,
			map[string]history.Counts{path: p.Counts})[path],
	}
	var out Curve
	switch p.Goal {
	case history.GoalRetention:
		out, err = u.retention(ctx, run, now, p, at, unseen)
	case history.GoalDate:
		out, err = u.date(ctx, run, now, p, at, unseen)
	default:
		out, err = u.minutes(ctx, run, now, p, at, unseen)
	}
	if err != nil {
		return Curve{}, err
	}
	out.Decks = mine
	out.Cards = len(under)
	out.Overdue = history.Overdue(u.Day, at, now)
	out.Unbegun = unseen
	return out, nil
}

// pointing is how many of the vault's decks name the preset at path. Asked is
// what has already been worked out from the cards standing.
func (u Curves) pointing(
	ctx context.Context, v domain.Vault, reading *Reading, path string, asked map[string]bool,
) (int, error) {
	if u.Standings.Notes == nil {
		out := 0
		for _, points := range asked {
			if points {
				out++
			}
		}
		return out, nil
	}

	decks, err := u.Standings.Notes.OfType(ctx, v.ID, domain.TypeDeck)
	if err != nil {
		return 0, err
	}
	out := 0
	for _, deck := range decks {
		points, held := asked[deck]
		if !held {
			p, err := reading.Of(ctx, v, deck)
			if err != nil {
				return 0, err
			}
			points = p.Path == path
		}
		if points {
			out++
		}
	}
	return out, nil
}

// minutes is the curve of how long a day of review runs.
//
// It runs from a short day to twice what carrying the whole load costs, so the
// place where the load is carried stands inside it. Each place is the sitting a
// person would sit down to now, which is the day the deck screen offers.
func (u Curves) minutes(
	ctx context.Context, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) (Curve, error) {
	free := p
	free.MinutesADay = int(history.MinutesADayBounds.Most)
	load, err := run.Run(ctx, now, free, at, unseen)
	if err != nil {
		return Curve{}, err
	}

	out := Curve{Goal: history.GoalMinutes, Now: Nowhere, Suggested: Nowhere}
	top := ceiling(carried(load), float64(p.MinutesADay))
	for i := range Points {
		out.Grid = append(out.Grid, math.Round(top*float64(i+1)/Points))
	}
	// The day the preset keeps is a place of the grid, so what is drawn under
	// the mark is drawn for the setting the person is standing at.
	standing(out.Grid, float64(p.MinutesADay), 0, len(out.Grid)-1)
	for _, minutes := range out.Grid {
		one := p
		one.MinutesADay = int(minutes)
		ran, err := run.Run(ctx, now, one, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		out.At = append(out.At, sitting(ran))
	}

	out.Now = Mark{At: nearest(out.Grid, float64(p.MinutesADay)), Value: float64(p.MinutesADay)}
	// What is suggested is the shortest day that asks everything the day holds:
	// the minutes stop closing it, and the material is what runs out. A load
	// nothing on the grid carries is suggested at the longest day on it.
	out.Suggested = Mark{At: len(out.Grid) - 1, Value: out.Grid[len(out.Grid)-1]}
	for i, one := range out.At {
		if one.Closed == history.ClosedNothing {
			out.Suggested = Mark{At: i, Value: out.Grid[i]}
			break
		}
	}
	return out, nil
}

// retention is the curve of how much is asked of memory.
//
// A higher target is shorter intervals and more reviews, so under a budget that
// binds it is also fewer cards kept up with.
//
// Each place is the load over the days projected, which is what the target costs
// week after week. Today is no part of it: the schedules a day opens with are
// the same whatever target is chosen, and what a target changes it changes from
// tomorrow on.
func (u Curves) retention(
	ctx context.Context, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) (Curve, error) {
	out := Curve{Goal: history.GoalRetention, Now: Nowhere, Suggested: Nowhere}
	least, most := history.RetentionBounds.Least, history.RetentionBounds.Most
	for i := range Points {
		out.Grid = append(out.Grid, least+(most-least)*float64(i)/float64(Points-1))
	}
	// The target the preset asks for is a place of the grid, so what is drawn
	// under the mark is drawn for it. The two ends are what memory allows, and
	// no setting displaces them.
	standing(out.Grid, p.Retention, 1, len(out.Grid)-2)

	for _, share := range out.Grid {
		one, asks := p, run
		one.Retention = share
		asks.By = u.at(share)
		ran, err := asks.Run(ctx, now, one, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		place := point(ran)
		// The day the whole material stands learned is not carried here. This
		// control moves the scheduler itself: the day a card passes an interval
		// is the reviews it takes times the space between them, and the reviews
		// are a whole number, so the day steps up wherever the range wants one
		// more of them and falls away between the steps. What a target buys over
		// the run is the share learned, beside it.
		place.Learns = history.LearnsUnasked
		out.At = append(out.At, place)
	}

	out.Now = Mark{At: nearest(out.Grid, p.Retention), Value: p.Retention}
	// Nothing is suggested. What a target leaves in the head climbs to the top
	// of the range, so a mark on the most of it would stand at the far end every
	// time and say only to ask for as much as memory allows.
	return out, nil
}

// date is the curve of getting through the material by a day.
//
// Each day of it carries what a day of review has to run to be through by then,
// and what the budget the preset keeps gets through by then.
//
// The range stands on the vault and not on the day the file names, so a person
// can always give themselves longer than they have. It begins tomorrow, because
// a date of today is no period at all, and reaches whichever is further off:
// twice as far as the day named, or the day the material would be through at
// one card a day, which is the slowest a day of review goes.
func (u Curves) date(
	ctx context.Context, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) (Curve, error) {
	out := Curve{Goal: history.GoalDate, Now: Nowhere, Suggested: Nowhere}
	open := u.Day.Ends(now).AddDate(0, 0, -1)
	by := p.By.Format(history.Named)
	if p.By.IsZero() || by < u.Day.Names(open) {
		return out, nil
	}
	// How far off the day the file names is, counting the day holding now as
	// none. A day further off than the projection reaches stands nowhere on the
	// range, and the range is drawn as far as it goes so that a person can see
	// where their day fell off it.
	named := 0
	for day := open; u.Day.Names(day) < by; day = day.AddDate(0, 0, 1) {
		named++
		if named > MostAhead {
			named = Nowhere.At
			break
		}
	}

	first := 1
	last := min(MostAhead, max(2*named, unseen, first))
	if named < 0 {
		last = MostAhead
	}
	run.Days = last + 1

	// Each place of the range is one day named, run at the pace that day sets:
	// the material spread over the days up to it. Everything read off a place
	// comes from that one run, so the height of the curve and the mark on it are
	// one answer.
	//
	// Every place runs the same horizon, however near its own day is, so what it
	// says about a backlog is the same question answered on every goal and not
	// one asked over as many days as the place stands off.
	for _, step := range naming(spread(last-first+1, Points), named-first) {
		day := first + step
		aiming, asks := p, run
		aiming.By = open.AddDate(0, 0, day)
		asks.Days = max(day+1, history.Ahead)
		ran, err := asks.Run(ctx, now, aiming, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		// A day at none of the load is no sitting at all, so what a day of
		// review holds is read off the first day this run admits.
		opening, sitting := ran.Sitting()
		out.Grid = append(out.Grid, float64(day))
		out.Days = append(out.Days, u.Day.Names(aiming.By))
		one := Point{
			// What it costs is what the days up to that one spend, and the days
			// past it are no part of getting through by it.
			Minutes: costing(ran.Spent[:day+1], ran.Admitted[:day+1]),
			// A place of this range is read on the day it names. Past that day
			// the preset schedules nothing, so a debt read off the end of the
			// horizon is a debt nobody was asked to pay.
			Owed:     ran.Backlog[day],
			Retained: ran.Retained[day],
			Through:  ran.Through[day],
			Enough:   reached(ran, day, ran.Short),
			Met:      reached(ran, day, ran.Short),
			Short:    ran.Short,
			Closed:   history.ClosedPaused,
			Clears:   ran.Clears,
			Learned:  ran.Learned,
			Learns:   ran.Learns,
			Backlog:  ran.Backlog,
		}
		if sitting {
			one.Reviews, one.Closed = float64(ran.Load[opening]), ran.Closed[opening]
		}
		out.At = append(out.At, one)
	}

	// The day the file names is a place of the grid, so what stands under the
	// mark is worked out for that day.
	if named >= 0 {
		out.Now = Mark{
			At:    nearest(out.Grid, float64(named)),
			Value: float64(named),
			Day:   u.Day.Names(open.AddDate(0, 0, named)),
		}
	}
	// What is suggested is the soonest day the material can be learned by:
	// nothing out of reach on it and the pace through the whole of it. Where the
	// preset keeps minutes, the soonest such day whose cost fits them, which is
	// being through it without changing the day a person sits to.
	if p.MinutesADay > 0 {
		for i, one := range out.At {
			if learns(one) && one.Minutes <= float64(p.MinutesADay) {
				out.Suggested = Mark{At: i, Value: out.Grid[i], Day: out.Days[i]}
				return out, nil
			}
		}
	}
	// A preset keeping no minutes, and a range no day of which fits them, are
	// suggested the soonest day that gets there at whatever it costs. A range no
	// day of which gets there points at none.
	for i, one := range out.At {
		if learns(one) {
			out.Suggested = Mark{At: i, Value: out.Grid[i], Day: out.Days[i]}
			return out, nil
		}
	}
	return out, nil
}

// learns reports whether the whole material stands learned on this day: no card
// face out of reach of it, and the pace it sets through every one of them.
func learns(one Point) bool { return one.Short == 0 && one.Enough }

// reached reports whether every card face that can be learned by this day of a
// run stands learned on it. Short is how many cannot be, whatever the pace.
func reached(p history.Projection, day, short int) bool {
	if p.Faces == 0 {
		return true
	}
	return p.Through[day] >= float64(p.Faces-short)/float64(p.Faces)
}

// point is a projection as one place of a curve, at the load it carries over
// the days the preset admits.
func point(p history.Projection) Point {
	return Point{
		Reviews:  p.ReviewsADay,
		Minutes:  p.MinutesADay,
		Retained: p.Retained[len(p.Retained)-1],
		Owed:     p.Owed,
		Through:  p.Through[len(p.Through)-1],
		Closed:   closing(p),
		Short:    p.Short,
		Clears:   p.Clears,
		Learned:  p.Learned,
		Learns:   p.Learns,
		Backlog:  p.Backlog,
	}
}

// closing is what closed the first day the preset admits. A preset admitting no
// day is closed by the pause.
func closing(p history.Projection) history.Closed {
	day, any := p.Sitting()
	if !any {
		return history.ClosedPaused
	}
	return p.Closed[day]
}

// sitting is a projection as one place of a curve, at the first day of it the
// preset admits: the next sitting a person will sit down to.
//
// It is one real day of the run, worked out by the arithmetic the deck screen
// runs, so the count here is the count that sitting hands a person. A preset
// admitting no day at all holds no sitting, and stands at nothing.
func sitting(p history.Projection) Point {
	out := point(p)
	day, any := p.Sitting()
	if !any {
		return out
	}
	out.Reviews, out.Minutes = float64(p.Load[day]), p.Spent[day].Minutes()
	return out
}

// spread is up to as many places as are wanted, evenly over the days, and
// always both ends of them.
func spread(days, places int) []int {
	if days <= places {
		out := make([]int, days)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := make([]int, places)
	for i := range out {
		out[i] = int(math.Round(float64(i) * float64(days-1) / float64(places-1)))
	}
	return out
}

// naming puts one place of the range on the grid, in place of the place
// nearest it. The two ends stand: a range begins tomorrow and reaches as far as
// it reaches, whatever day the file names.
//
// The day the preset aims at is the day a person is looking at, so the point
// under the mark is worked out for that day and not for the day beside it.
func naming(steps []int, at int) []int {
	if at < 0 || len(steps) < 3 {
		return steps
	}
	near := 1
	for i := 2; i < len(steps)-1; i++ {
		if abs(steps[i]-at) < abs(steps[near]-at) {
			near = i
		}
	}
	if at > steps[0] && at < steps[len(steps)-1] {
		steps[near] = at
	}
	return steps
}

// standing puts the value the preset holds on the grid, in place of the place
// of it nearest that value, so what is drawn under the mark is drawn for the
// setting the person is standing at.
//
// First and last are the places a value may take: a range whose ends say what
// the setting may be at all keeps them.
func standing(grid []float64, value float64, first, last int) {
	if first < 0 || last >= len(grid) || first > last {
		return
	}
	at := first
	for i := first; i <= last; i++ {
		if math.Abs(grid[i]-value) < math.Abs(grid[at]-value) {
			at = i
		}
	}
	grid[at] = value
}

// abs is how far a whole number stands from nothing.
func abs(one int) int {
	if one < 0 {
		return -one
	}
	return one
}

// carried is how long the first day the preset admits took, which is what
// carrying the whole load costs on the next sitting.
func carried(p history.Projection) float64 {
	day, any := p.Sitting()
	if !any {
		return 0
	}
	return p.Spent[day].Minutes()
}

// ceiling is how far a curve of minutes runs: twice what carrying the whole
// load costs, and never less than a short day or more than a day holds.
func ceiling(load, keeping float64) float64 {
	top := math.Ceil(2 * math.Max(load, keeping))
	return math.Min(math.Max(top, LeastCeiling), history.MinutesADayBounds.Most)
}

// nearest is the place of the grid a value falls at, and -1 for a value outside
// it.
func nearest(grid []float64, value float64) int {
	if len(grid) == 0 || value < grid[0] || value > grid[len(grid)-1] {
		return -1
	}
	at := 0
	for i, one := range grid {
		if math.Abs(one-value) < math.Abs(grid[at]-value) {
			at = i
		}
	}
	return at
}

// at is the scheduler asking for a share of the cards to come back.
func (u Curves) at(retention float64) history.Scheduler {
	if u.At != nil {
		return u.At(retention)
	}
	return history.NewFSRSAt(retention)
}

func (u Curves) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

// costing is how long a day of review runs over these days, in minutes.
//
// A day the preset does not admit is no sitting at all and takes no part: a
// week of five days runs its minutes over five days.
func costing(spent []time.Duration, admitted []bool) float64 {
	var all time.Duration
	days := 0
	for i, one := range spent {
		if i < len(admitted) && !admitted[i] {
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
