package flashcards

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

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
	Now     port.Clock
	// By is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	By func(retention float64) review.Scheduler
	// Cores is how many places of a curve are worked out at once. A build
	// holding none works one place at a time.
	Cores int
}

// NewProjectCurve is what a preset's one control is projected through: what
// stands in the vault, where the answers have left each card face, which preset
// each deck is scheduled by, where one day of review gives way to the next, and
// what time it is.
func NewProjectCurve(
	faces ListCardFaces, schedules Schedules, presets Presets,
	day review.Day, now port.Clock,
) ProjectCurve {
	return ProjectCurve{
		CardFaces: faces, Schedules: schedules, Presets: presets, Day: day, Now: now,
	}
}

// checkGoal is what is wrong with the value the goal moves, and is nil where
// the value stands inside its bounds. A goal of a date names a day and no
// number.
func checkGoal(p review.Preset) error {
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
	if bounds.Contains(value) {
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
) (review.Curve, error) {
	// The value the goal steers is written into the grid, and a grid runs only
	// between the bounds of it.
	if err := checkGoal(p); err != nil {
		return review.Curve{}, err
	}
	scheduled, err := u.getScheduledDecks(ctx, v, path)
	if err != nil {
		return review.Curve{}, err
	}
	faces := u.CardFaces.GetFaces(ctx, v, scheduled)
	held, err := Log{Stores: u.Schedules.Logs}.Read(ctx, v)
	if err != nil {
		return review.Curve{}, err
	}
	// A deck is asked once which preset schedules it, however many card faces
	// it holds, and a preset note is opened once however many decks name it.
	reading := u.Presets.Reading()
	asks, err := u.Schedules.getAssignmentFrom(ctx, v, reading, faces)
	if err != nil {
		return review.Curve{}, err
	}
	schedules := u.Schedules.getSchedules(held, asks)

	decks := make(map[string]bool)
	at := make(map[review.CardFaceID]review.Schedule)
	// The card faces this preset schedules, which is what it is costed from.
	under := make(map[review.CardFaceID]string)
	unseen := 0
	for _, one := range faces {
		mine, asked := decks[one.Deck]
		if !asked {
			held, err := reading.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return review.Curve{}, err
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
	mine, err := u.countPointingDecks(ctx, v, reading, path, scheduled, decks)
	if err != nil {
		return review.Curve{}, err
	}

	cost, costed := review.GetCostUnder(u.Schedules.By, held.Answers, under)[path]
	if !costed {
		cost = review.DefaultCost
	}
	now := u.Now()
	// The projection is run by the scheduler this preset asks for, which is the
	// one its cards are scheduled by, and it opens on the day a person is
	// already partway through.
	run := review.Simulation{
		By: u.getScheduler(p.Retention), Day: u.Day, Cost: cost,
		Spent: review.GetSpentUnder(u.Day, u.Day.GetName(now), held.Answers, under,
			map[string]review.BudgetUnit{path: p.Counts})[path],
	}
	out, err := run.Curve(ctx, now, p, at, unseen, u.getScheduler, u.places)
	if err != nil {
		return review.Curve{}, err
	}
	out.Stops = p.GetOverallStopReason(u.Day, now)
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
func (u ProjectCurve) getScheduledDecks(ctx context.Context, v domain.Vault, path string) ([]string, error) {
	decks, err := u.CardFaces.Decks(ctx, v)
	if err != nil || path == "" || u.Presets.Links == nil {
		return decks, err
	}

	at, err := u.Presets.Links.Backlinks(ctx, v.ID, path)
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

// countPointingDecks is how many of these decks name the preset at path. Asked
// is what has already been worked out from the cards standing.
func (u ProjectCurve) countPointingDecks(
	ctx context.Context, v domain.Vault, reading *PresetReads, path string,
	decks []string, asked map[string]bool,
) (int, error) {
	out := 0
	for _, deck := range decks {
		points, held := asked[deck]
		if !held {
			p, err := reading.GetForDeck(ctx, v, deck)
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

// getScheduler is the scheduler asking for a share of the cards to come back.
func (u ProjectCurve) getScheduler(retention float64) review.Scheduler {
	if u.By != nil {
		return u.By(retention)
	}
	return review.NewFSRSAt(retention)
}
