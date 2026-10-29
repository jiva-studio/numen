package projection

import (
	"context"
	"math"
	"time"
)

// Points is how many places a curve is worked out at. A goal of minutes and a
// goal of retention run at exactly this many, and a goal of a date whose range
// is shorter runs at one place a day.
const Points = 25

// MostAhead is how far ahead a goal of a date is projected. A day beyond this is
// outside the range, and the curve is drawn to the end of the range.
const MostAhead = 5 * 365

// Curve represents preset evaluation across the goal range.
type Curve struct {
	Goal Goal
	// Grid is the value of the goal at each place: minutes for minutes_a_day, a
	// share of cards for retention, and days from today for by_date.
	Grid []float64
	// Days names the day of each place, and is filled for a goal of a date.
	Days []string
	// Points are what the preset evaluates to at each place of the grid.
	Points []Point
	// Now is the current preset point.
	Now CurvePoint
	// Suggested is the recommended point on the curve.
	// For retention goals, this is OutsideGrid.
	Suggested CurvePoint
	// Stops indicates why settings schedule nothing, empty when scheduling cards.
	Stops StopReason
	// Decks is the count of decks scheduled by this preset.
	Decks int
	// Cards is the total count of review cards in those decks.
	Cards int
	// Overdue is the count of review cards past due.
	// Unbegun is the count of review cards with no review history.
	Overdue int
	Unbegun int
}

// Point is what a preset evaluates to at one place of the grid.
//
// Reviews is counted in review cards and Minutes in the showings they take, so a
// review card the day comes back to costs its minutes and is the one card.
//
// The meaning of the two depends on the goal. Under a goal of minutes they are
// the first day of the run the preset admits: the next session that will be
// run. Under a goal of retention they are the load over the days the preset
// admits.
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
	// The number of review cards due on the last day the projection ran:
	// Ahead days off under a goal of minutes or of retention, and the day the
	// place names under a goal of a date.
	Owed int
	// Share is the fraction of cards learned by this day.
	// IsEnough reports whether all learnable cards are learned.
	Share    float64
	IsEnough bool
	// Short is how many review cards cannot be learned by this day whatever the
	// pace, which is the rule wanting more days than the day leaves them.
	Short int
	// Closed is every budget that closed the day here, in the words the preset
	// writes them in, and is empty where the material itself ran out.
	Closed BudgetNames
	// Clears is how many days of review at this place it takes before nothing
	// is overdue. A curve standing over nothing overdue clears in none, and a
	// place whose pace never gets there is NeverClears.
	Clears int
	// Learned is how many review cards stand learned today at this place, and
	// Learns how many days of review it takes before all of them do. A place
	// whose horizon ends with one still to learn is NeverLearns, and one with no
	// such day to name is LearnsUnasked.
	Learned int
	Learns  int
	// Backlog is how many review cards stand overdue at the end of each day
	// projected at this place, one entry a day over the whole horizon. It runs
	// over days and not over the goal's range.
	Backlog []int
}

// CurvePoint is a point on the curve.
type CurvePoint struct {
	// Index is the grid index, or -1 when outside the grid.
	Index int
	// Value is the goal value at this point.
	Value float64
	// Day is the date string for date goals.
	Day string
}

// OutsideGrid indicates a point falling outside the grid.
var OutsideGrid = CurvePoint{Index: -1, Value: 0, Day: ""}

// createCurve is the curve of a goal before any place is computed.
func createCurve(goal Goal) Curve {
	return Curve{
		Goal: goal, Grid: nil, Days: nil, Points: nil, Now: OutsideGrid, Suggested: OutsideGrid,
		Stops: "", Decks: 0, Cards: 0, Overdue: 0, Unbegun: 0,
	}
}

// curveBuilder holds simulation state for computing a curve.
type curveBuilder struct {
	run          Simulation
	now          time.Time
	preset       Preset
	schedules    map[ReviewCardID]Schedule
	unbegunCount int
	scheduler    func(retention float64) Scheduler
	places       func(count int, each func(index int) error) error
}

// Curve evaluates a preset over the goal range using simulation.
func (s Simulation) Curve(
	ctx context.Context, now time.Time, p Preset, schedules map[ReviewCardID]Schedule,
	unbegunCount int, scheduler func(retention float64) Scheduler,
	places func(count int, each func(index int) error) error,
) (Curve, error) {
	if scheduler == nil {
		scheduler = func(retention float64) Scheduler { return NewFSRSAt(retention) }
	}
	if places == nil {
		places = func(count int, each func(index int) error) error {
			for i := range count {
				if err := each(i); err != nil {
					return err
				}
			}
			return nil
		}
	}
	d := curveBuilder{run: s, now: now, preset: p, schedules: schedules, unbegunCount: unbegunCount, scheduler: scheduler, places: places}
	switch p.Goal {
	case GoalRetention:
		return d.retention(ctx)
	case GoalDate:
		return d.date(ctx)
	case GoalMinutes:
		return d.minutes(ctx)
	default:
		return d.minutes(ctx)
	}
}

// minutes computes the curve for daily review duration goals.
func (d curveBuilder) minutes(ctx context.Context) (Curve, error) {
	run, now, p, pos, unreviewed := d.run, d.now, d.preset, d.schedules, d.unbegunCount

	free := p
	free.MinutesADay = int(MinutesADayBounds.Max)
	load, err := run.Run(ctx, now, free, pos, unreviewed)
	if err != nil {
		return Curve{}, err
	}

	out := createCurve(GoalMinutes)
	top := getMinutesUpperBound(getSessionMinutes(load), float64(p.MinutesADay))
	for i := range Points {
		out.Grid = append(out.Grid, math.Round(top*float64(i+1)/Points))
	}
	// The day the preset keeps is a place of the grid.
	snap(out.Grid, float64(p.MinutesADay), 0, len(out.Grid)-1)
	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.GetDurationDays() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := d.places(len(out.Grid), func(i int) error {
		one := p
		one.MinutesADay = int(out.Grid[i])
		ran, err := run.Run(ctx, now, one, pos, unreviewed)
		if err != nil {
			return err
		}
		place := session(ran)
		if place.Learns, err = getDaysToLearnWithoutForgetting(ctx, run, now, one, pos, unreviewed); err != nil {
			return err
		}
		out.Points[i] = place
		return nil
	}); err != nil {
		return Curve{}, err
	}

	out.Now = CurvePoint{Index: findNearest(out.Grid, float64(p.MinutesADay)), Value: float64(p.MinutesADay), Day: ""}
	// The suggested value is the shortest day that covers everything the day holds:
	// the minutes stop closing it, and the material runs out. A load that no grid
	// value carries is suggested at the longest day on the grid.
	out.Suggested = CurvePoint{Index: len(out.Grid) - 1, Value: out.Grid[len(out.Grid)-1], Day: ""}
	for i, one := range out.Points {
		if len(one.Closed) == 0 {
			out.Suggested = CurvePoint{Index: i, Value: out.Grid[i], Day: ""}
			break
		}
	}
	return out, nil
}

// retention is the curve of the memory demand per target.
//
// A higher target means shorter intervals and more reviews, so under a binding
// budget it also keeps up with fewer cards.
//
// Each place is the load over the projected days, which is what the target
// costs week after week. Today is excluded: the schedules a day opens with are
// the same for every target, and a target changes them from tomorrow on.
func (d curveBuilder) retention(ctx context.Context) (Curve, error) {
	run, now, p, pos, unreviewed := d.run, d.now, d.preset, d.schedules, d.unbegunCount

	out := createCurve(GoalRetention)
	least, most := RetentionBounds.Min, RetentionBounds.Max
	for i := range Points {
		out.Grid = append(out.Grid, least+(most-least)*float64(i)/float64(Points-1))
	}
	// The preset's target is a place of the grid. The two ends are fixed by what
	// memory allows, and no setting displaces them.
	snap(out.Grid, p.Retention, 1, len(out.Grid)-2)

	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	run.Retains = []int{run.GetDurationDays() - 1}
	out.Points = make([]Point, len(out.Grid))
	if err := d.places(len(out.Grid), func(i int) error {
		one, asks := p, run
		one.Retention = out.Grid[i]
		asks.By = d.scheduler(out.Grid[i])
		ran, err := asks.Run(ctx, now, one, pos, unreviewed)
		if err != nil {
			return err
		}
		place := point(ran)
		if place.Learns, err = getDaysToLearnWithoutForgetting(ctx, asks, now, one, pos, unreviewed); err != nil {
			return err
		}
		out.Points[i] = place
		return nil
	}); err != nil {
		return Curve{}, err
	}

	out.Now = CurvePoint{Index: findNearest(out.Grid, p.Retention), Value: p.Retention, Day: ""}
	return out, nil
}

// date computes the curve of completing reviews by a specified target date.
func getNamedOffset(d Day, open time.Time, targetDate string) int {
	named := 0
	for day := open; d.GetName(day) < targetDate; day = day.AddDate(0, 0, 1) {
		named++
		if named > MostAhead {
			return OutsideGrid.Index
		}
	}
	return named
}

// createDatePoint constructs the curve point for a simulated day.
func createDatePoint(ran Projection, day int) Point {
	back, _ := ran.Retained.GetShare(day)
	opening, session := ran.Session()
	one := Point{
		Reviews:  0,
		Minutes:  ran.minutesADay(day),
		Owed:     ran.Backlog[day],
		Retained: back,
		Share:    ran.Through[day],
		IsEnough: ran.isAllLearned(day),
		Short:    ran.Short,
		Closed:   BudgetNames{ClosedPaused},
		Clears:   ran.Clears,
		Learned:  ran.Learned,
		Learns:   ran.Learns,
		Backlog:  ran.Backlog,
	}
	if session {
		one.Reviews, one.Closed = float64(ran.Reviewed[opening]), ran.Closed[opening]
	}
	return one
}

// findSuggestedDate returns the earliest learned point within budget or the earliest learned point overall.
func findSuggestedDate(points []Point, grid []float64, days []string, minutesADay int) CurvePoint {
	if minutesADay > 0 {
		for i, one := range points {
			if isLearned(one) && one.Minutes <= float64(minutesADay) {
				return CurvePoint{Index: i, Value: grid[i], Day: days[i]}
			}
		}
	}
	for i, one := range points {
		if isLearned(one) {
			return CurvePoint{Index: i, Value: grid[i], Day: days[i]}
		}
	}
	return OutsideGrid
}
func (d curveBuilder) date(ctx context.Context) (Curve, error) {
	run, now, p, pos, unreviewedCount := d.run, d.now, d.preset, d.schedules, d.unbegunCount

	out := createCurve(GoalDate)
	open := run.Day.GetStart(now)
	delta := p.By.Format(DayLayout)
	if p.By.IsZero() || delta < run.Day.GetName(open) {
		return out, nil
	}
	named := getNamedOffset(run.Day, open, delta)

	first := 1
	last := min(MostAhead, max(2*named, unreviewedCount, first))
	if named < 0 {
		last = MostAhead
	}
	run.Days = last + 1

	// Each place of the range is one day named, run at the pace that day sets:
	// the material spread over the days up to it. Everything read off a place
	// comes from that one run.
	//
	// A place runs a horizon of Ahead days, or up to its own day when that is
	// further off.
	steps := setNearestStep(pickEvenlySpacedDays(last-first+1, Points), named-first)
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
		ran, err := asks.Run(ctx, now, aiming, pos, unreviewedCount)
		if err != nil {
			return err
		}
		// A day at none of the load is no session at all, so what a day of
		// review holds is read off the first day this run admits.
		out.Grid[i] = float64(day)
		out.Days[i] = run.Day.GetName(aiming.By)
		out.Points[i] = createDatePoint(ran, day)
		return nil
	}); err != nil {
		return Curve{}, err
	}

	// The day the file names is a place of the grid, so what is placed under the
	// place is worked out for that day.
	if named >= 0 {
		out.Now = CurvePoint{
			Index: findNearest(out.Grid, float64(named)),
			Value: float64(named),
			Day:   run.Day.GetName(open.AddDate(0, 0, named)),
		}
	}
	out.Suggested = findSuggestedDate(out.Points, out.Grid, out.Days, p.MinutesADay)
	return out, nil
}

// getDaysToLearnWithoutForgetting calculates days needed to learn all cards assuming full recall.
func getDaysToLearnWithoutForgetting(
	ctx context.Context, run Simulation, now time.Time, p Preset,
	pos map[ReviewCardID]Schedule, unbegunCount int,
) (int, error) {
	run.Recalls = GetFullRecall
	// The day the material is learned is all this run is read for.
	run.Retains = nil
	ran, err := run.Run(ctx, now, p, pos, unbegunCount)
	if err != nil {
		return 0, err
	}
	return ran.Learns, nil
}

// isLearned reports whether all material is learned on this day.
func isLearned(one Point) bool { return one.Short == 0 && one.IsEnough }

// point is a projection as one place of a curve, at the load it carries over
// the days the preset admits.
func point(p Projection) Point {
	// A place is read on the last day of its run, and that is the day the run
	// works the returning share out on.
	back, _ := p.Retained.GetShare(p.Days - 1)
	return Point{
		Reviews: p.ReviewsADay,
		Minutes: p.MinutesADay,
		// A goal of minutes and a goal of retention set no pace at a day, so no
		// place of theirs falls short of one.
		IsEnough: true,
		Retained: back,
		Owed:     p.Owed,
		Share:    p.Through[len(p.Through)-1],
		Closed:   getSessionClosed(p),
		Short:    p.Short,
		Clears:   p.Clears,
		Learned:  p.Learned,
		Learns:   p.Learns,
		Backlog:  p.Backlog,
	}
}

// getSessionClosed is what closed the first day the preset admits. A preset
// admitting no day is closed by the pause.
func getSessionClosed(p Projection) BudgetNames {
	day, isAdmitted := p.Session()
	if !isAdmitted {
		return BudgetNames{ClosedPaused}
	}
	return p.Closed[day]
}

// session converts a projection to a Point for the first admitted day.
func session(p Projection) Point {
	out := point(p)
	day, isAdmitted := p.Session()
	if !isAdmitted {
		return out
	}
	out.Reviews, out.Minutes = float64(p.Reviewed[day]), p.Spent[day].Minutes()
	return out
}

// getSessionMinutes returns the duration in minutes for the first admitted day.
func getSessionMinutes(p Projection) float64 {
	day, isAdmitted := p.Session()
	if !isAdmitted {
		return 0
	}
	return p.Spent[day].Minutes()
}
