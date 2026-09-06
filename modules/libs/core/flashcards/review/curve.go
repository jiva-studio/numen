package review

import (
	"context"
	"math"
	"time"
)

// Points is how many places a curve is worked out at. A goal of minutes and a
// goal of retention run at exactly this many, and a goal of a date whose range
// is shorter runs at one place a day.
const Points = 25

// MostAhead is how far ahead a goal of a date is projected. A day further off
// than this stands nowhere on the range, and the curve is drawn to the end of
// the range.
const MostAhead = 5 * 365

// Curve is what the one control of a preset comes to over the whole range of
// its goal. The grid, what stands at each place of it, and the two places
// pointed at are all here.
type Curve struct {
	Goal Goal
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
	Stops StopReason
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
// minutes they are the first day of the run the preset admits: the next session
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
	Closed BudgetNames
	// Clears is how many days of review at this place it takes before nothing
	// is overdue. A curve standing over nothing overdue clears in none, and a
	// place whose pace never gets there is NeverClears.
	Clears int
	// Learned is how many card faces stand learned today at this place, and
	// Learns how many days of review it takes before all of them do. A place
	// whose horizon ends with one still to learn is NeverLearns, and one with no
	// such day to name is LearnsUnasked.
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

// drawing is one curve being worked out: the run every place of it is projected
// with, and what stands the same at every place.
type drawing struct {
	run    Simulation
	now    time.Time
	preset Preset
	at     map[CardFaceID]Schedule
	unseen int
	by     func(retention float64) Scheduler
	places func(count int, each func(at int) error) error
}

// Curve is what the one control of a preset comes to over the whole range of
// its goal, worked out by running this projection at every place of the range.
//
// The map is where the answers have left every card face the preset schedules,
// and unseen is how many of its card faces nobody has answered, as Run reads
// them. By is the scheduler asking for a share of the cards to come back, and
// is asked again at each place of a goal of retention.
//
// Places runs the places of the range and is where a caller says how many of
// them run at once; a caller handing over none runs them one at a time.
func (s Simulation) Curve(
	ctx context.Context, now time.Time, p Preset, at map[CardFaceID]Schedule,
	unseen int, by func(retention float64) Scheduler,
	places func(count int, each func(at int) error) error,
) (Curve, error) {
	if by == nil {
		by = func(retention float64) Scheduler { return NewFSRSAt(retention) }
	}
	if places == nil {
		places = func(count int, each func(at int) error) error {
			for i := range count {
				if err := each(i); err != nil {
					return err
				}
			}
			return nil
		}
	}
	d := drawing{run: s, now: now, preset: p, at: at, unseen: unseen, by: by, places: places}
	switch p.Goal {
	case GoalRetention:
		return d.retention(ctx)
	case GoalDate:
		return d.date(ctx)
	default:
		return d.minutes(ctx)
	}
}

// minutes is the curve of how long a day of review runs.
//
// It runs from a short day to twice what carrying the whole load costs, so the
// place where the load is carried stands inside it. Each place is the session a
// person would sit down to now, which is the day the deck screen offers.
func (d drawing) minutes(ctx context.Context) (Curve, error) {
	run, now, p, at, unseen := d.run, d.now, d.preset, d.at, d.unseen

	free := p
	free.MinutesADay = int(MinutesADayBounds.Most)
	load, err := run.Run(ctx, now, free, at, unseen)
	if err != nil {
		return Curve{}, err
	}

	out := Curve{Goal: GoalMinutes, Now: Nowhere, Suggested: Nowhere}
	top := ceiling(carried(load), float64(p.MinutesADay))
	for i := range Points {
		out.Grid = append(out.Grid, math.Round(top*float64(i+1)/Points))
	}
	// The day the preset keeps is a place of the grid.
	snap(out.Grid, float64(p.MinutesADay), 0, len(out.Grid)-1)
	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.Covers() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := d.places(len(out.Grid), func(i int) error {
		one := p
		one.MinutesADay = int(out.Grid[i])
		ran, err := run.Run(ctx, now, one, at, unseen)
		if err != nil {
			return err
		}
		place := session(ran)
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
func (d drawing) retention(ctx context.Context) (Curve, error) {
	run, now, p, at, unseen := d.run, d.now, d.preset, d.at, d.unseen

	out := Curve{Goal: GoalRetention, Now: Nowhere, Suggested: Nowhere}
	least, most := RetentionBounds.Least, RetentionBounds.Most
	for i := range Points {
		out.Grid = append(out.Grid, least+(most-least)*float64(i)/float64(Points-1))
	}
	// The target the preset asks for is a place of the grid. The two ends are
	// what memory allows, and no setting displaces them.
	snap(out.Grid, p.Retention, 1, len(out.Grid)-2)

	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.Covers() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := d.places(len(out.Grid), func(i int) error {
		one, asks := p, run
		one.Retention = out.Grid[i]
		asks.By = d.by(out.Grid[i])
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
func (d drawing) date(ctx context.Context) (Curve, error) {
	run, now, p, at, unseen := d.run, d.now, d.preset, d.at, d.unseen

	out := Curve{Goal: GoalDate, Now: Nowhere, Suggested: Nowhere}
	open := run.Day.Opens(now)
	by := p.By.Format(Named)
	if p.By.IsZero() || by < run.Day.Names(open) {
		return out, nil
	}
	// How far off the day the file names is, counting the day holding now as
	// none. A day further off than the projection reaches stands nowhere on the
	// range, and the range is drawn as far as it goes.
	named := 0
	for day := open; run.Day.Names(day) < by; day = day.AddDate(0, 0, 1) {
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
	if err := d.places(len(steps), func(i int) error {
		day := first + steps[i]
		aiming, asks := p, run
		aiming.By = open.AddDate(0, 0, day)
		asks.Days = max(day+1, Ahead)
		// A place of this range is read on the day it names, and that is the day
		// the run works the returning share out on.
		asks.Retains = []int{day}
		ran, err := asks.Run(ctx, now, aiming, at, unseen)
		if err != nil {
			return err
		}
		back, _ := ran.Retained.On(day)
		// A day at none of the load is no session at all, so what a day of
		// review holds is read off the first day this run admits.
		opening, session := ran.Session()
		out.Grid[i] = float64(day)
		out.Days[i] = run.Day.Names(aiming.By)
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
			Closed:   BudgetNames{ClosedPaused},
			Clears:   ran.Clears,
			Learned:  ran.Learned,
			Learns:   ran.Learns,
			Backlog:  ran.Backlog,
		}
		if session {
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
			Day:   run.Day.Names(open.AddDate(0, 0, named)),
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

// learnt is how many days of review it takes before the whole material stands
// learned, projected on the assumption that nothing is forgotten.
//
// The day it names is the day a person reaches if the answers go well. It is
// the one figure drawn under that assumption, and the run and the arithmetic
// are those of every figure beside it.
func learnt(
	ctx context.Context, run Simulation, now time.Time, p Preset,
	at map[CardFaceID]Schedule, unseen int,
) (int, error) {
	run.Recalls = NothingForgotten
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
func reached(p Projection, day, short int) bool {
	if p.Faces == 0 {
		return true
	}
	return p.Through[day] >= float64(p.Faces-short)/float64(p.Faces)
}

// point is a projection as one place of a curve, at the load it carries over
// the days the preset admits.
func point(p Projection) Point {
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

// closing is what closed the first day the preset admits. A preset admession no
// day is closed by the pause.
func closing(p Projection) BudgetNames {
	day, any := p.Session()
	if !any {
		return BudgetNames{ClosedPaused}
	}
	return p.Closed[day]
}

// session is a projection as one place of a curve, at the first day of it the
// preset admits: the next session a person will sit down to.
//
// It is one real day of the run, worked out by the arithmetic the deck screen
// runs, so the count here is the count that session hands a person. A preset
// admession no day at all holds no session, and stands at nothing.
func session(p Projection) Point {
	out := point(p)
	day, any := p.Session()
	if !any {
		return out
	}
	out.Reviews, out.Minutes = float64(p.Faced[day]), p.Spent[day].Minutes()
	return out
}

// carried is how long the first day the preset admits took, which is what
// carrying the whole load costs on the next session.
func carried(p Projection) float64 {
	day, any := p.Session()
	if !any {
		return 0
	}
	return p.Spent[day].Minutes()
}

// costing is how long a day of review runs over these days, in minutes.
//
// A day the preset does not admit is no session at all and takes no part: a
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
