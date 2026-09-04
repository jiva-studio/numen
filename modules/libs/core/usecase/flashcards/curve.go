package flashcards

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// Points is how many places a curve is worked out at. A goal of minutes and a
// goal of retention run at exactly this many, and a goal of a date whose range
// is shorter runs at one place a day.
const Points = 25

// LeastCeiling is the shortest day a curve of minutes runs to.
const LeastCeiling = 60

// MostAhead is how far ahead a goal of a date is projected. A day further off
// than this stands nowhere on the range, and the curve is drawn to the end of
// the range.
const MostAhead = 5 * 365

// Curve is what the one control of a preset comes to over the whole range of
// its goal. The grid, what stands at each place of it, and the two places
// pointed at are all here.
type Curve struct {
	Goal review.Goal
	// Grid is the value of the goal at each place: minutes for minutes_a_day, a
	// share of cards for retention, and days from today for by_date.
	Grid []float64
	// Days names the day of each place, and is filled for a goal of a date.
	Days []string
	// Points are what the preset comes to at each place of the grid.
	Points []Point
	// Now is where the preset stands.
	Now Place
	// Suggested is the place worth pointing at: under a goal of minutes the
	// shortest day the clock no longer cuts short, and under a goal of a date
	// the soonest day the material is learned by at a cost the minutes the
	// preset keeps allow. A goal of retention has none, and stands at Nowhere.
	Suggested Place
	// Stops is why the settings this curve was drawn under schedule nothing,
	// and is empty where they schedule something. It is asked of the settings
	// the request carried.
	Stops review.StopReason
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
// Reviews is counted in card faces and Minutes in the showings they take, so a
// face the day comes back to costs its minutes and is the one card.
//
// What the two are the height of is the goal's own question. Under a goal of
// minutes they are the first day of the run the preset admits: the next sitting
// a person will actually sit down to. Under a goal of retention they are the
// load over the days the preset admits.
//
// A goal of a date reads Reviews off that first day, and fills Minutes with
// what getting through the material by the day the place names costs, over the
// days of review up to it.
type Point struct {
	Reviews float64
	Minutes float64
	// Retained is the share of the material that comes back on the last day
	// projected, which is what the load beside it buys.
	Retained float64
	// Owed is the card faces standing owed on the last day the projection ran:
	// Ahead days off under a goal of minutes or of retention, and the day the
	// place names under a goal of a date.
	Owed int
	// Share is the share of the material learned by this day, and Enough is
	// whether the pace this place sets learns every card face that can be
	// learned by it. A goal keeping no such account stands at true.
	Share  float64
	Enough bool
	// Short is how many card faces cannot be learned by this day whatever the
	// pace, which is the rule wanting more days than the day leaves them.
	Short int
	// Closed is every budget that closed the day here, in the words the preset
	// writes them in, and is empty where the material itself ran out.
	Closed review.BudgetNames
	// Clears is how many days of review at this place it takes before nothing
	// is overdue. A curve standing over nothing overdue clears in none, and a
	// place whose pace never gets there is review.NeverClears.
	Clears int
	// Learned is how many card faces stand learned today at this place, and
	// Learns how many days of review it takes before all of them do. A place
	// whose horizon ends with one still to learn is review.NeverLearns, and one
	// with no such day to name is review.LearnsUnasked.
	Learned int
	Learns  int
	// Backlog is how many card faces stand overdue at the end of each day
	// projected at this place, one entry a day over the whole horizon. It runs
	// over days and not over the goal's range.
	Backlog []int
}

// Place is one place on the curve worth pointing at.
type Place struct {
	// Index is the place of the grid, and is -1 when the value falls outside it.
	Index int
	// Value is the goal's value at the place, in the units of the grid. What the
	// preset stands at need not sit on the grid.
	Value float64
	// Day is the day at the place, and is filled for a goal of a date.
	Day string
}

// Nowhere is a place that falls outside the grid.
var Nowhere = Place{Index: -1}

// ProjectCurve is the simulator behind the one control of a preset.
//
// It reads the vault's answers once and projects them forward at every place of
// the goal's range. Nothing here writes.
type ProjectCurve struct {
	CardFaces ListCardFaces
	Schedules Schedules
	// Presets says which preset each deck is scheduled by. A build holding no
	// links projects every deck of the vault.
	Presets Presets
	Day     review.Day
	Now     func() time.Time
	// By is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	By func(retention float64) review.Scheduler
	// Cores is how many places of a curve are worked out at once. It is a fact
	// about the machine, so it is given here rather than asked of the runtime,
	// and a build holding none works one place at a time.
	Cores int
}

// steered is what is wrong with the value the goal moves, and is nil where the
// value stands inside its bounds. A goal of a date names a day and no number.
func steered(p review.Preset) error {
	var value float64
	var bounds review.Bounds
	var key string
	switch p.Goal {
	case review.GoalMinutes:
		value, bounds, key = float64(p.MinutesADay), review.MinutesADayBounds, minutesADayKey
	case review.GoalRetention:
		value, bounds, key = p.Retention, review.RetentionBounds, retentionKey
	default:
		return nil
	}
	if bounds.Holds(value) {
		return nil
	}
	return fmt.Errorf("%w: %s %g is outside %g to %g",
		ErrOutOfBounds, key, value, bounds.Least, bounds.Most)
}

// Execute is the curve of the goal this preset steers.
//
// The preset is the settings the curve is drawn under, and need not be what its
// note holds. Path is the note the preset stands in and names the decks it
// schedules, and a preset standing in no note schedules the decks that name
// none.
func (u ProjectCurve) Execute(
	ctx context.Context, v domain.Vault, path string, p review.Preset,
) (Curve, error) {
	// The value the goal steers is written into the grid, and a grid runs only
	// between the bounds of it.
	if err := steered(p); err != nil {
		return Curve{}, err
	}
	scheduled, err := u.scheduled(ctx, v, path)
	if err != nil {
		return Curve{}, err
	}
	standing := u.CardFaces.Of(ctx, v, scheduled)
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
	schedules := u.Schedules.worked(held, asks)

	decks := make(map[string]bool)
	at := make(map[review.CardFaceID]review.Schedule)
	// The card faces this preset schedules, which is what it is costed from.
	under := make(map[review.CardFaceID]string)
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
		under[one.ID] = path
		s, answered := schedules[one.ID]
		if !answered {
			unseen++
			continue
		}
		at[one.ID] = s
	}

	// How many decks this preset schedules, counted over every deck that could
	// name it: a deck of no cards points at its preset like any other.
	mine, err := u.pointing(ctx, v, reading, path, scheduled, decks)
	if err != nil {
		return Curve{}, err
	}

	cost, costed := review.CostedUnder(u.Schedules.By, held.Answers, under)[path]
	if !costed {
		cost = review.DefaultCost
	}
	now := u.now()
	// The projection is run by the scheduler this preset asks for, which is the
	// one its cards are scheduled by, and it opens on the day a person is
	// already partway through.
	run := review.Simulation{
		By: u.at(p.Retention), Day: u.Day, Cost: cost,
		Spent: review.Sat(u.Day, u.Day.Names(now), held.Answers, under,
			map[string]review.Counts{path: p.Counts})[path],
	}
	var out Curve
	switch p.Goal {
	case review.GoalRetention:
		out, err = u.retention(ctx, run, now, p, at, unseen)
	case review.GoalDate:
		out, err = u.date(ctx, run, now, p, at, unseen)
	default:
		out, err = u.minutes(ctx, run, now, p, at, unseen)
	}
	if err != nil {
		return Curve{}, err
	}
	out.Stops = p.Stops(u.Day, now)
	out.Decks = mine
	out.Cards = len(under)
	out.Overdue = review.Overdue(u.Day, at, now)
	out.Unbegun = unseen
	return out, nil
}

// scheduled is the decks a curve of the preset at path is worked out over.
//
// A deck names its preset with an entry of its `links:` block, so what points
// at that note is what the preset could schedule, and the rest of the vault is
// left unread. Which of them the preset does schedule is the reading's answer:
// a deck naming two presets is scheduled by the first.
//
// A preset standing in no note is the defaults, and nothing points at those.
// The decks they schedule are the decks naming no preset, which is a question
// only the decks answer: every one of them is read, and the curve of the
// defaults pays for the whole vault.
func (u ProjectCurve) scheduled(ctx context.Context, v domain.Vault, path string) ([]string, error) {
	decks, err := u.CardFaces.Decks(ctx, v)
	if err != nil || path == "" || u.Presets.Links == nil {
		return decks, err
	}

	at, err := u.Presets.Links.Backlinks(ctx, string(v.ID), path)
	if err != nil {
		return nil, fmt.Errorf("what points at %s: %w", path, err)
	}
	naming := make(map[string]bool, len(at))
	for _, link := range at {
		if link.Type == LinkType {
			naming[link.From] = true
		}
	}

	out := make([]string, 0, len(naming))
	for _, deck := range decks {
		if naming[deck] {
			out = append(out, deck)
		}
	}
	return out, nil
}

// pointing is how many of these decks name the preset at path. Asked is what
// has already been worked out from the cards standing.
func (u ProjectCurve) pointing(
	ctx context.Context, v domain.Vault, reading *PresetReads, path string,
	decks []string, asked map[string]bool,
) (int, error) {
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
func (u ProjectCurve) minutes(
	ctx context.Context, run review.Simulation, now time.Time, p review.Preset,
	at map[review.CardFaceID]review.Schedule, unseen int,
) (Curve, error) {
	free := p
	free.MinutesADay = int(review.MinutesADayBounds.Most)
	load, err := run.Run(ctx, now, free, at, unseen)
	if err != nil {
		return Curve{}, err
	}

	out := Curve{Goal: review.GoalMinutes, Now: Nowhere, Suggested: Nowhere}
	top := ceiling(carried(load), float64(p.MinutesADay))
	for i := range Points {
		out.Grid = append(out.Grid, math.Round(top*float64(i+1)/Points))
	}
	// The day the preset keeps is a place of the grid.
	standing(out.Grid, float64(p.MinutesADay), 0, len(out.Grid)-1)
	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.Covers() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := u.places(len(out.Grid), func(i int) error {
		one := p
		one.MinutesADay = int(out.Grid[i])
		ran, err := run.Run(ctx, now, one, at, unseen)
		if err != nil {
			return err
		}
		place := sitting(ran)
		if place.Learns, err = learnt(ctx, run, now, one, at, unseen); err != nil {
			return err
		}
		out.Points[i] = place
		return nil
	}); err != nil {
		return Curve{}, err
	}

	out.Now = Place{Index: nearest(out.Grid, float64(p.MinutesADay)), Value: float64(p.MinutesADay)}
	// What is suggested is the shortest day that asks everything the day holds:
	// the minutes stop closing it, and the material is what runs out. A load
	// nothing on the grid carries is suggested at the longest day on it.
	out.Suggested = Place{Index: len(out.Grid) - 1, Value: out.Grid[len(out.Grid)-1]}
	for i, one := range out.Points {
		if len(one.Closed) == 0 {
			out.Suggested = Place{Index: i, Value: out.Grid[i]}
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
func (u ProjectCurve) retention(
	ctx context.Context, run review.Simulation, now time.Time, p review.Preset,
	at map[review.CardFaceID]review.Schedule, unseen int,
) (Curve, error) {
	out := Curve{Goal: review.GoalRetention, Now: Nowhere, Suggested: Nowhere}
	least, most := review.RetentionBounds.Least, review.RetentionBounds.Most
	for i := range Points {
		out.Grid = append(out.Grid, least+(most-least)*float64(i)/float64(Points-1))
	}
	// The target the preset asks for is a place of the grid. The two ends are
	// what memory allows, and no setting displaces them.
	standing(out.Grid, p.Retention, 1, len(out.Grid)-2)

	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.Covers() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := u.places(len(out.Grid), func(i int) error {
		one, asks := p, run
		one.Retention = out.Grid[i]
		asks.By = u.at(out.Grid[i])
		ran, err := asks.Run(ctx, now, one, at, unseen)
		if err != nil {
			return err
		}
		place := point(ran)
		if place.Learns, err = learnt(ctx, asks, now, one, at, unseen); err != nil {
			return err
		}
		out.Points[i] = place
		return nil
	}); err != nil {
		return Curve{}, err
	}

	out.Now = Place{Index: nearest(out.Grid, p.Retention), Value: p.Retention}
	// A goal of retention suggests nothing, and its place stands at Nowhere.
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
func (u ProjectCurve) date(
	ctx context.Context, run review.Simulation, now time.Time, p review.Preset,
	at map[review.CardFaceID]review.Schedule, unseen int,
) (Curve, error) {
	out := Curve{Goal: review.GoalDate, Now: Nowhere, Suggested: Nowhere}
	open := u.Day.Opens(now)
	by := p.By.Format(review.Named)
	if p.By.IsZero() || by < u.Day.Names(open) {
		return out, nil
	}
	// How far off the day the file names is, counting the day holding now as
	// none. A day further off than the projection reaches stands nowhere on the
	// range, and the range is drawn as far as it goes.
	named := 0
	for day := open; u.Day.Names(day) < by; day = day.AddDate(0, 0, 1) {
		named++
		if named > MostAhead {
			named = Nowhere.Index
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
	// comes from that one run.
	//
	// A place runs a horizon of Ahead days, and of its own day where that stands
	// further off.
	steps := naming(spread(last-first+1, Points), named-first)
	out.Grid = make([]float64, len(steps))
	out.Days = make([]string, len(steps))
	out.Points = make([]Point, len(steps))
	if err := u.places(len(steps), func(i int) error {
		day := first + steps[i]
		aiming, asks := p, run
		aiming.By = open.AddDate(0, 0, day)
		asks.Days = max(day+1, review.Ahead)
		// A place of this range is read on the day it names, and that is the day
		// the run works the returning share out on.
		asks.Retains = []int{day}
		ran, err := asks.Run(ctx, now, aiming, at, unseen)
		if err != nil {
			return err
		}
		back, _ := ran.Retained.On(day)
		// A day at none of the load is no sitting at all, so what a day of
		// review holds is read off the first day this run admits.
		opening, sitting := ran.Sitting()
		out.Grid[i] = float64(day)
		out.Days[i] = u.Day.Names(aiming.By)
		one := Point{
			// What it costs is what the days up to that one spend, and the days
			// past it are no part of getting through by it.
			Minutes: costing(ran.Spent[:day+1], ran.Admitted[:day+1]),
			// A place of this range is read on the day it names. Past that day
			// the preset schedules nothing, so a debt read off the end of the
			// horizon is a debt nobody was asked to pay.
			Owed:     ran.Backlog[day],
			Retained: back,
			Share:    ran.Through[day],
			Enough:   reached(ran, day, ran.Short),
			Short:    ran.Short,
			Closed:   review.BudgetNames{review.ClosedPaused},
			Clears:   ran.Clears,
			Learned:  ran.Learned,
			Learns:   ran.Learns,
			Backlog:  ran.Backlog,
		}
		if sitting {
			one.Reviews, one.Closed = float64(ran.Faced[opening]), ran.Closed[opening]
		}
		out.Points[i] = one
		return nil
	}); err != nil {
		return Curve{}, err
	}

	// The day the file names is a place of the grid, so what stands under the
	// place is worked out for that day.
	if named >= 0 {
		out.Now = Place{
			Index: nearest(out.Grid, float64(named)),
			Value: float64(named),
			Day:   u.Day.Names(open.AddDate(0, 0, named)),
		}
	}
	// What is suggested is the soonest day the material can be learned by:
	// nothing out of reach on it and the pace through the whole of it. Where the
	// preset keeps minutes, it is the soonest such day whose cost fits them.
	if p.MinutesADay > 0 {
		for i, one := range out.Points {
			if learns(one) && one.Minutes <= float64(p.MinutesADay) {
				out.Suggested = Place{Index: i, Value: out.Grid[i], Day: out.Days[i]}
				return out, nil
			}
		}
	}
	// A preset keeping no minutes, and a range no day of which fits them, are
	// suggested the soonest day that gets there at whatever it costs. A range no
	// day of which gets there points at none.
	for i, one := range out.Points {
		if learns(one) {
			out.Suggested = Place{Index: i, Value: out.Grid[i], Day: out.Days[i]}
			return out, nil
		}
	}
	return out, nil
}

// places works out every place of a grid, each of them alongside the others.
//
// No place reads another's answer: a run is given the schedules the answers
// have already produced, reads them and nothing else, and writes only what it
// hands back. Each answer is put down at the place it belongs to.
//
// A place that fails is left to the places beside it, and the error handed back
// is the earliest place's. A request nobody is waiting for is ended by the run
// of each place reading the context it was given.
func (u ProjectCurve) places(count int, each func(at int) error) error {
	failed := make([]error, count)
	room := make(chan struct{}, max(1, u.Cores))
	var running sync.WaitGroup
	for at := range count {
		room <- struct{}{}
		running.Add(1)
		go func() {
			defer running.Done()
			defer func() { <-room }()
			failed[at] = each(at)
		}()
	}
	running.Wait()

	for _, err := range failed {
		if err != nil {
			return err
		}
	}
	return nil
}

// learnt is how many days of review it takes before the whole material stands
// learned, projected on the assumption that nothing is forgotten.
//
// The day it names is the day a person reaches if the answers go well. It is
// the one figure drawn under that assumption, and the run and the arithmetic
// are those of every figure beside it.
func learnt(
	ctx context.Context, run review.Simulation, now time.Time, p review.Preset,
	at map[review.CardFaceID]review.Schedule, unseen int,
) (int, error) {
	run.Recalls = review.NothingForgotten
	// The day the material is learned is all this run is read for.
	run.Retains = nil
	ran, err := run.Run(ctx, now, p, at, unseen)
	if err != nil {
		return 0, err
	}
	return ran.Learns, nil
}

// learns reports whether the whole material stands learned on this day: no card
// face out of reach of it, and the pace it sets through every one of them.
func learns(one Point) bool { return one.Short == 0 && one.Enough }

// reached reports whether every card face that can be learned by this day of a
// run stands learned on it. Short is how many cannot be, whatever the pace.
func reached(p review.Projection, day, short int) bool {
	if p.Faces == 0 {
		return true
	}
	return p.Through[day] >= float64(p.Faces-short)/float64(p.Faces)
}

// point is a projection as one place of a curve, at the load it carries over
// the days the preset admits.
func point(p review.Projection) Point {
	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	back, _ := p.Retained.On(p.Days - 1)
	return Point{
		Reviews: p.ReviewsADay,
		Minutes: p.MinutesADay,
		// A goal of minutes and a goal of retention set no pace at a day, so no
		// place of theirs falls short of one.
		Enough:   true,
		Retained: back,
		Owed:     p.Owed,
		Share:    p.Through[len(p.Through)-1],
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
func closing(p review.Projection) review.BudgetNames {
	day, any := p.Sitting()
	if !any {
		return review.BudgetNames{review.ClosedPaused}
	}
	return p.Closed[day]
}

// sitting is a projection as one place of a curve, at the first day of it the
// preset admits: the next sitting a person will sit down to.
//
// It is one real day of the run, worked out by the arithmetic the deck screen
// runs, so the count here is the count that sitting hands a person. A preset
// admitting no day at all holds no sitting, and stands at nothing.
func sitting(p review.Projection) Point {
	out := point(p)
	day, any := p.Sitting()
	if !any {
		return out
	}
	out.Reviews, out.Minutes = float64(p.Faced[day]), p.Spent[day].Minutes()
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
// The point under the place is worked out for the day the preset aims at.
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
// of it nearest that value, so what is drawn under the place is drawn for the
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
func carried(p review.Projection) float64 {
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
	return math.Min(math.Max(top, LeastCeiling), review.MinutesADayBounds.Most)
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
func (u ProjectCurve) at(retention float64) review.Scheduler {
	if u.By != nil {
		return u.By(retention)
	}
	return review.NewFSRSAt(retention)
}

func (u ProjectCurve) now() time.Time {
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
