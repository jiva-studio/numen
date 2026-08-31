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
	// Now is where the preset stands, and Suggested is what is suggested.
	Now       Mark
	Suggested Mark
	// Decks is how many decks are scheduled by this preset. Zero is a preset no
	// deck points at, and every place of the curve stands at zero with it.
	Decks int
	// Cards is how many card faces stand in those decks. Zero is a preset with
	// nothing to schedule, and every place of the curve stands at zero with it.
	Cards int
}

// Point is what a preset comes to at one place of the grid.
//
// A goal of a date fills Minutes with what getting through the material by that
// day costs, and Through and Enough with what the budget the preset keeps gets
// through by it. The rest stands at zero there.
type Point struct {
	// Reviews and Minutes are the daily load.
	Reviews float64
	Minutes float64
	// Retained is the share of the material that comes back.
	Retained float64
	// Owed is the backlog the budget did not carry.
	Owed int
	// Through is the share of the material got through by this day, and Enough
	// is whether the budget the preset keeps gets through all of it. Met is
	// whether any budget does.
	Through float64
	Enough  bool
	Met     bool
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
	schedules := projected(held, asks)

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
	// The projection is run by the scheduler this preset asks for, which is the
	// one its cards are scheduled by.
	run := history.Simulation{By: u.at(p.Retention), Day: u.Day, Cost: cost}
	now := u.now()
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
// place where the load is carried stands inside it.
func (u Curves) minutes(
	ctx context.Context, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) (Curve, error) {
	free := p
	free.MinutesADay = 0
	load, err := run.Run(ctx, now, free, at, unseen)
	if err != nil {
		return Curve{}, err
	}

	out := Curve{Goal: history.GoalMinutes, Now: Nowhere, Suggested: Nowhere}
	top := ceiling(load.MinutesADay, float64(p.MinutesADay))
	for i := range Points {
		out.Grid = append(out.Grid, math.Round(top*float64(i+1)/Points))
	}
	for _, minutes := range out.Grid {
		one := p
		one.MinutesADay = int(minutes)
		ran, err := run.Run(ctx, now, one, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		out.At = append(out.At, point(ran))
	}

	out.Now = Mark{At: nearest(out.Grid, float64(p.MinutesADay)), Value: float64(p.MinutesADay)}
	// What is suggested is the shortest day that pays the whole debt. A load
	// nothing on the grid carries is suggested at the longest day on it.
	out.Suggested = Mark{At: len(out.Grid) - 1, Value: out.Grid[len(out.Grid)-1]}
	for i, one := range out.At {
		if one.Owed == 0 {
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
func (u Curves) retention(
	ctx context.Context, run history.Simulation, now time.Time, p history.Preset,
	at map[history.CardFace]history.Schedule, unseen int,
) (Curve, error) {
	out := Curve{Goal: history.GoalRetention, Now: Nowhere, Suggested: Nowhere}
	least, most := history.RetentionBounds.Least, history.RetentionBounds.Most
	for i := range Points {
		out.Grid = append(out.Grid, least+(most-least)*float64(i)/float64(Points-1))
	}

	for _, share := range out.Grid {
		one, asks := p, run
		one.Retention = share
		asks.By = u.at(share)
		ran, err := asks.Run(ctx, now, one, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		out.At = append(out.At, point(ran))
	}

	out.Now = Mark{At: nearest(out.Grid, p.Retention), Value: p.Retention}
	// What is suggested is the target that leaves the most of the material in
	// the head. Two targets that leave the same is the cheaper of them.
	best := 0
	for i, one := range out.At {
		if one.Retained > out.At[best].Retained {
			best = i
		}
	}
	out.Suggested = Mark{At: best, Value: out.Grid[best]}
	return out, nil
}

// date is the curve of getting through the material by a day.
//
// Each day of it carries what a day of review has to run to be through by then,
// and what the budget the preset keeps gets through by then.
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
	days := 1
	for day := open; u.Day.Names(day) < by; day = day.AddDate(0, 0, 1) {
		days++
		if days > MostAhead {
			return out, nil
		}
	}
	run.Days = days

	// The whole range is walked once at each budget, and the day a budget gets
	// through the material is read out of the walk. The budgets are walked from
	// the longest day down, so what stands against a day at the end is the
	// shortest day that got through by it.
	standing, err := run.Run(ctx, now, p, at, unseen)
	if err != nil {
		return Curve{}, err
	}
	free := p
	free.MinutesADay = 0
	carrying, err := run.Run(ctx, now, free, at, unseen)
	if err != nil {
		return Curve{}, err
	}
	top := ceiling(carrying.MinutesADay, float64(p.MinutesADay))
	// A day no budget on the range gets through the material by needs more than
	// the range explores, so it stands at the top of it. The walk below only
	// ever lowers a day, and the minutes a day needed fall as the days grow.
	needs := make([]float64, days)
	for day := range needs {
		needs[day] = top
	}
	met := make([]bool, days)
	for i := Points; i >= 1; i-- {
		one := p
		one.MinutesADay = int(math.Round(top * float64(i) / Points))
		ran, err := run.Run(ctx, now, one, at, unseen)
		if err != nil {
			return Curve{}, err
		}
		for day, share := range ran.Through {
			if share >= 1 {
				needs[day], met[day] = float64(one.MinutesADay), true
			}
		}
	}

	for _, day := range spread(days, Points) {
		out.Grid = append(out.Grid, float64(day))
		out.Days = append(out.Days, u.Day.Names(open.AddDate(0, 0, day)))
		out.At = append(out.At, Point{
			Minutes: needs[day],
			Through: standing.Through[day],
			Enough:  standing.Through[day] >= 1,
			Met:     met[day],
		})
	}

	last := len(out.Grid) - 1
	out.Now = Mark{At: last, Value: out.Grid[last], Day: out.Days[last]}
	// What is suggested is the first day the budget the preset keeps gets
	// through the material by. A budget that never does is suggested the first
	// day any budget does.
	for i, one := range out.At {
		if one.Enough {
			out.Suggested = Mark{At: i, Value: out.Grid[i], Day: out.Days[i]}
			return out, nil
		}
	}
	for i, one := range out.At {
		if one.Met {
			out.Suggested = Mark{At: i, Value: out.Grid[i], Day: out.Days[i]}
			return out, nil
		}
	}
	return out, nil
}

// point is a projection as one place of a curve.
func point(p history.Projection) Point {
	return Point{
		Reviews:  p.ReviewsADay,
		Minutes:  p.MinutesADay,
		Retained: p.Retained,
		Owed:     p.Owed,
		Through:  p.Through[len(p.Through)-1],
	}
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
