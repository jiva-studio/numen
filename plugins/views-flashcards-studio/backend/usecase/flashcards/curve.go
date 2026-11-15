package flashcards

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/presets"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/projection"
)

// Curve is the simulated progression of a preset across its target range.
type Curve = projection.Curve

type CurvePoint = projection.CurvePoint

// Point is one place of a simulated curve.
type Point = projection.Point

// Points is how many places a curve is worked out at.
const Points = projection.Points

// Ahead is how many days a projection runs when no length is given.
const Ahead = projection.Ahead

// MostAhead is how far ahead a goal of a date is projected.
const MostAhead = projection.MostAhead

func GetOutsideGrid() CurvePoint { return projection.GetOutsideGrid() }

// ProjectCurve is the simulator behind the one control of a preset.
//
// It reads the vault's answers once and projects them forward at every place of
// the goal's range. Nothing here writes.
type ProjectCurve struct {
	reviewCards ListReviewCards
	schedules   Schedules
	presets     presets.OpenReads
	day         review.Day
	now         ports.Clock
	By          func(retention float64) review.Scheduler
	Cores       int
}

// NewProjectCurve is what a preset's one control is projected through: what
// each deck is scheduled by, where one day of review gives way to the next, and
// what time it is.
func NewProjectCurve(
	reviewCards ListReviewCards, schedules Schedules, presets presets.OpenReads,
	day review.Day, now ports.Clock,
) ProjectCurve {
	return ProjectCurve{
		reviewCards: reviewCards, schedules: schedules, presets: presets, day: day, now: now,
		By:    nil,
		Cores: 0,
	}
}

func checkGoal(p review.Preset) error {
	var value float64
	var bounds review.Bounds
	var key string
	switch p.Goal {
	case review.GoalMinutes:
		value, bounds, key = float64(p.MinutesADay), review.GetMinutesADayBounds(), presets.MinutesADayKey
	case review.GoalRetention:
		value, bounds, key = p.Retention, review.GetRetentionBounds(), presets.RetentionKey
	case review.GoalDate:
		// gomutants:disable-next-line BRANCH_CASE reason="Zero value and zero bounds satisfy Contains when GoalDate exits the switch."
		return nil
	default:
		return nil
	}
	if bounds.Contains(value) {
		return nil
	}
	return fmt.Errorf("%w: %s %g is outside %g to %g",
		presets.ErrOutOfBounds, key, value, bounds.Min, bounds.Max)
}

// Execute is the curve of the goal this preset steers.
//
// The preset is the settings the curve is drawn presets, and need not be what its
// schedules, and a preset standing in no note schedules the decks that name
// none.
func (u ProjectCurve) Execute(
	ctx context.Context, v domain.Vault, path string, p review.Preset,
) (Curve, error) {
	// The value the goal steers is written into the grid, and a grid runs only
	// between the bounds of it.
	if err := checkGoal(p); err != nil {
		return Curve{}, err
	}
	scheduled, err := u.getScheduledDecks(ctx, v, path)
	if err != nil {
		return Curve{}, err
	}
	reviewCards := u.reviewCards.GetReviewCards(ctx, v, scheduled)
	target, err := Log{Stores: u.schedules.logs}.Read(ctx, v)
	if err != nil {
		return Curve{}, err
	}
	// A deck is queried once for the preset that schedules it, however many review
	// cards it holds, and a preset note is opened once however many decks name it.
	reads := u.presets.Execute()
	asks, err := u.schedules.getAssignmentFrom(ctx, v, reads, reviewCards)
	if err != nil {
		return Curve{}, err
	}
	schedules := u.schedules.getSchedules(target, asks)

	mine, err := u.readScheduledCards(ctx, v, reads, path, reviewCards, schedules)
	// gomutants:disable-next-line BRANCH_IF reason="PresetReads cache is already populated for all reviewCard decks by getAssignmentFrom."
	if err != nil {
		return Curve{}, err
	}
	decks, pos, presets, unbegunCount := mine.decks, mine.schedules, mine.presetMap, mine.unreviewed

	// How many decks this preset schedules, counted over every deck that could
	// name it: a deck of no cards points at its preset like any other.
	pointing, err := u.countPointingDecks(ctx, v, reads, path, scheduled, decks)
	if err != nil {
		return Curve{}, err
	}

	cost, costed := review.GetCostUnder(u.schedules.scheduler, target.Answers, presets)[path]
	if !costed {
		cost = review.GetDefaultCost()
	}
	now := u.now.Now()
	// The projection runs with the scheduler this preset requests, which is the one
	// that schedules its cards, and it starts on the current day with the answers
	// already given.
	run := projection.Simulation{
		By: u.getScheduler(p.Retention), Day: u.day, Cost: cost,
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Non-positive simulation Days defaults to HorizonDays."
		Days:    0,
		Retains: nil,
		Recalls: nil,
		Spent: review.GetSpentUnder(u.day, u.day.GetName(now), target.Answers, presets,
			map[string]review.BudgetUnit{path: p.Counts})[path],
	}
	out, err := run.Curve(ctx, now, p, projection.CurveInputs{
		Schedules: pos, UnbegunCount: unbegunCount, Scheduler: u.getScheduler, Places: u.computePlaces,
	})
	if err != nil {
		return Curve{}, err
	}
	out.Stops = p.GetOverallStopReason(u.day, now)
	out.Decks = pointing
	out.Cards = len(presets)
	out.Overdue = projection.CountOverdue(u.day, pos, now)
	out.Unbegun = unbegunCount
	return out, nil
}

// scheduledCards is the review cards a preset schedules.
type scheduledCards struct {
	decks      map[string]bool
	schedules  map[review.CardID]review.Schedule
	presetMap  map[review.CardID]string
	unreviewed int
}

func (u ProjectCurve) readScheduledCards(
	ctx context.Context, v domain.Vault, reads *presets.Reads, path string,
	reviewCards []ReviewCard, schedules map[review.CardID]review.Schedule,
) (scheduledCards, error) {
	out := scheduledCards{
		decks:      make(map[string]bool),
		schedules:  make(map[review.CardID]review.Schedule),
		presetMap:  make(map[review.CardID]string),
		unreviewed: 0,
	}
	for _, one := range reviewCards {
		mine, request := out.decks[one.Deck]
		if !request {
			target, err := reads.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return scheduledCards{}, err
			}
			mine = target.Path == path
			out.decks[one.Deck] = mine
		}
		if !mine {
			continue
		}
		out.presetMap[one.ID] = path
		s, reply := schedules[one.ID]
		if !reply {
			out.unreviewed++
			continue
		}
		out.schedules[one.ID] = s
	}
	return out, nil
}

func (u ProjectCurve) getScheduledDecks(ctx context.Context, v domain.Vault, path string) ([]string, error) {
	decks, err := u.reviewCards.GetDeckPaths(ctx, v)
	reads := u.presets.Execute()
	if err != nil || path == "" || !reads.HasLinks() {
		return decks, err
	}

	pos, err := reads.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return nil, fmt.Errorf("what points at %s: %w", path, err)
	}
	naming := make(map[string]bool, len(pos))
	for _, link := range pos {
		if link.Type == presets.LinkType {
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

func (u ProjectCurve) countPointingDecks(
	ctx context.Context, v domain.Vault, reads *presets.Reads, path string,
	decks []string, requested map[string]bool,
) (int, error) {
	out := 0
	for _, deck := range decks {
		points, target := requested[deck]
		if !target {
			p, err := reads.GetForDeck(ctx, v, deck)
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

func (u ProjectCurve) computePlaces(count int, each func(index int) error) error {
	failed := make([]error, count)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="Worker semaphore capacity affects concurrency bound only."
	room := make(chan struct{}, max(1, u.Cores))
	var running sync.WaitGroup
	for item := range count {
		room <- struct{}{}
		running.Go(func() {
			defer func() { <-room }()
			failed[item] = each(item)
		})
	}
	running.Wait()

	for _, err := range failed {
		if err != nil {
			return err
		}
	}
	return nil
}

func (u ProjectCurve) getScheduler(retention float64) review.Scheduler {
	if u.By != nil {
		return u.By(retention)
	}
	return review.NewFSRSAt(retention)
}
