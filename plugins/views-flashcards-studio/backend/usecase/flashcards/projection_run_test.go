package flashcards_test

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/projection"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

// reviewRun is a projection run through the review scheduler, over review presets and schedules.
type reviewRun struct {
	By      review.Scheduler
	Day     review.Day
	Cost    review.AnswerCost
	Days    int
	Retains []int
	Recalls review.RecallChance
	Spent   review.Spent
}

func (r reviewRun) Run(
	ctx context.Context, now time.Time, p review.Preset, schedules map[review.CardID]review.Schedule, unbegun int,
) (projection.Projection, error) {
	run := projection.Simulation{
		By:      flashcards.NewProjectionScheduler(r.By, r.Day),
		Day:     r.Day,
		Cost:    flashcards.ConvertCostToProjection(r.Cost),
		Days:    r.Days,
		Retains: r.Retains,
		Recalls: nil,
		Spent:   flashcards.ConvertSpentToProjection(r.Spent),
	}
	if r.Recalls != nil {
		run.Recalls = func(c projection.Card, at time.Time) float64 { return r.Recalls(getSchedule(c), at) }
	}
	return run.Run(ctx, now, flashcards.ConvertPresetToSettings(p), flashcards.ConvertSchedulesToCards(schedules), unbegun)
}

func getSchedule(c projection.Card) review.Schedule {
	return review.Schedule{
		Due: c.Due, Last: c.Last, Reps: c.Reps, Lapses: c.Lapses, Stability: c.Stability, Difficulty: c.Difficulty, Phase: c.Phase,
	}
}

func getDaysToLearn(by review.Scheduler, d review.Day, p review.Preset, now time.Time) int {
	return projection.GetDaysToLearn(flashcards.NewProjectionScheduler(by, d), d, flashcards.ConvertPresetToSettings(p), now)
}

func countOverdue(d review.Day, schedules map[review.CardID]review.Schedule, now time.Time) int {
	return projection.CountOverdue(d, flashcards.ConvertSchedulesToCards(schedules), now)
}
