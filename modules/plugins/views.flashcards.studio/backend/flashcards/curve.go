package flashcards

import (
	"context"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// ProjectCurve is the simulator behind the one control of a preset.
//
// It reads the vault's answers once and projects them forward at every place of
// the goal's range. Nothing here writes.
type ProjectCurve struct {
	cardFaces ListCardFaces
	schedules Schedules
	// presets says which preset each deck is scheduled by. A build holding no
	// links projects every deck of the vault.
	presets Presets
	day     review.Day
	now     ports.Clock
	// By is the scheduler asking for a share of the cards to come back. A build
	// holding none reads FSRS.
	By func(retention float64) review.Scheduler
	// Cores is how many places of a curve are worked out at once. A build
	// holding none works one place at a time.
	Cores int
}

// NewProjectCurve is what a preset's one control is projected through: what
// each deck is scheduled by, where one day of review gives way to the next, and
// what time it is.
func NewProjectCurve(
	faces ListCardFaces, schedules Schedules, presets Presets,
	day review.Day, now ports.Clock,
) ProjectCurve {
	return ProjectCurve{
		cardFaces: faces, schedules: schedules, presets: presets, day: day, now: now,
	}
}

// checkGoal is what is wrong with the value the goal moves, and is nil where
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
// The preset is the settings the curve is drawn presets, and need not be what its
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
	faces := u.cardFaces.GetFaces(ctx, v, scheduled)
	target, err := Log{Stores: u.schedules.logs}.Read(ctx, v)
	if err != nil {
		return review.Curve{}, err
	}
	// A deck is asked once which preset schedules it, however many card faces
	// it holds, and a preset note is opened once however many decks name it.
	reads := u.presets.OpenReads()
	asks, err := u.schedules.getAssignmentFrom(ctx, v, reads, faces)
	if err != nil {
		return review.Curve{}, err
	}
	schedules := u.schedules.getSchedules(target, asks)

	mine, err := u.readScheduledFaces(ctx, v, reads, path, faces, schedules)
	if err != nil {
		return review.Curve{}, err
	}
	decks, pos, presets, unbegunCount := mine.decks, mine.schedules, mine.presetMap, mine.unreviewed

	// How many decks this preset schedules, counted over every deck that could
	// name it: a deck of no cards points at its preset like any other.
	pointing, err := u.countPointingDecks(ctx, v, reads, path, scheduled, decks)
	if err != nil {
		return review.Curve{}, err
	}

	cost, costed := review.GetCostUnder(u.schedules.scheduler, target.Answers, presets)[path]
	if !costed {
		cost = review.DefaultCost
	}
	now := u.now()
	// The projection is run by the scheduler this preset asks for, which is the
	// one its cards are scheduled by, and it opens on the day a person is
	// already partway through.
	run := review.Simulation{
		By: u.getScheduler(p.Retention), Day: u.day, Cost: cost,
		Spent: review.GetSpentUnder(u.day, u.day.GetName(now), target.Answers, presets,
			map[string]review.BudgetUnit{path: p.Counts})[path],
	}
	out, err := run.Curve(ctx, now, p, pos, unbegunCount, u.getScheduler, u.computePlaces)
	if err != nil {
		return review.Curve{}, err
	}
	out.Stops = p.GetOverallStopReason(u.day, now)
	out.Decks = pointing
	out.Cards = len(presets)
	out.Overdue = review.CountOverdue(u.day, pos, now)
	out.Unbegun = unbegunCount
	return out, nil
}

// scheduledFaces is the card faces a preset schedules.
type scheduledFaces struct {
	// decks says for each deck asked whether the preset schedules it.
	decks map[string]bool
	// schedules is the schedule of each face that has been answered.
	schedules map[review.CardFaceID]review.Schedule
	// presetMap is the faces the preset is costed from.
	presetMap map[review.CardFaceID]string
	// unseen is how many of the faces have not been answered.
	unreviewed int
}

// readScheduledFaces picks out of faces the ones the preset at path schedules.
func (u ProjectCurve) readScheduledFaces(
	ctx context.Context, v domain.Vault, reads *PresetReads, path string,
	faces []CardFace, schedules map[review.CardFaceID]review.Schedule,
) (scheduledFaces, error) {
	out := scheduledFaces{
		decks:     make(map[string]bool),
		schedules: make(map[review.CardFaceID]review.Schedule),
		presetMap: make(map[review.CardFaceID]string),
	}
	for _, one := range faces {
		mine, request := out.decks[one.Deck]
		if !request {
			target, err := reads.GetForDeck(ctx, v, one.Deck)
			if err != nil {
				return scheduledFaces{}, err
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

// scheduled is the decks a curve of the preset at path is worked out over.
//
// A deck names its preset with an entry of its `links:` block, so what points
// at that note is what the preset could schedule, and the rest of the vault is
// left unread. Which of them the preset does schedule is the answer of the reads:
// a deck naming two presets is scheduled by the first.
//
// A preset standing in no note is the defaults, and nothing points at those.
// The decks they schedule are the decks naming no preset, which is a question
// only the decks answer: every one of them is read, and the curve of the
// defaults pays for the whole vault.
func (u ProjectCurve) getScheduledDecks(ctx context.Context, v domain.Vault, path string) ([]string, error) {
	decks, err := u.cardFaces.GetDeckPaths(ctx, v)
	if err != nil || path == "" || u.presets.links == nil {
		return decks, err
	}

	pos, err := u.presets.links.GetBacklinks(ctx, v.ID, path)
	if err != nil {
		return nil, fmt.Errorf("what points at %s: %w", path, err)
	}
	naming := make(map[string]bool, len(pos))
	for _, link := range pos {
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
	ctx context.Context, v domain.Vault, reads *PresetReads, path string,
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

// computePlaces works out every place of a grid, each of them alongside the others.
//
// No place reads another's answer: a run is given the schedules the answers
// have already produced, reads them and nothing else, and writes only what it
// returns. Each answer is put down at the place it belongs to.
//
// A place that fails is left to the places beside it, and the error handed back
// is the earliest place's. A request nobody is waiting for is ended by the run
// of each place reading the context it was given.
func (u ProjectCurve) computePlaces(count int, each func(index int) error) error {
	failed := make([]error, count)
	room := make(chan struct{}, max(1, u.Cores))
	var running sync.WaitGroup
	for item := range count {
		room <- struct{}{}
		running.Add(1)
		go func() {
			defer running.Done()
			defer func() { <-room }()
			failed[item] = each(item)
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
