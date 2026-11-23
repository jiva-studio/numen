package flashcards

import (
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/projection"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// ProjectionScheduler answers what a projection asks of the review rules.
type ProjectionScheduler struct {
	by  review.Scheduler
	day review.Day
}

// NewProjectionScheduler places each due day on the day a review session places the card on.
func NewProjectionScheduler(by review.Scheduler, day review.Day) ProjectionScheduler {
	return ProjectionScheduler{by: by, day: day}
}

func (s ProjectionScheduler) GetEndings(c projection.Card, at time.Time) (recalled, forgotten projection.Card) {
	good, again := s.by.ComputeEndings(convertCardToSchedule(c), at)
	return convertScheduleToCard(good), convertScheduleToCard(again)
}

func (s ProjectionScheduler) GetRecall(c projection.Card, at time.Time) float64 {
	return review.GetModeledRecall(convertCardToSchedule(c), at)
}

func (s ProjectionScheduler) IsLearned(p projection.Settings, c projection.Card, at time.Time) bool {
	return convertSettingsToPreset(p).IsLearned(convertCardToSchedule(c), at)
}

func (s ProjectionScheduler) GetAllowance(
	p projection.Settings, open time.Time, spent projection.Spent, unbegunCards, daysToLearn int,
) projection.Allowance {
	a := convertSettingsToPreset(p).GetAllowance(s.day, open, review.Spent{
		Answered: spent.Answered, New: spent.New, Reviews: spent.Reviews, Took: spent.Took,
	}, unbegunCards, convertDaysToLearn(daysToLearn))
	return projection.Allowance{
		New:     a.New,
		Reviews: a.Reviews,
		Minutes: a.Minutes,
		Limits: projection.Limits{
			New:     convertBudgetName(a.Limits.New),
			Reviews: convertBudgetName(a.Limits.Reviews),
			Minutes: convertBudgetName(a.Limits.Minutes),
		},
		Backlog:  a.Backlog,
		IsPaused: a.IsPaused(),
	}
}

func (s ProjectionScheduler) NewLoad() projection.Load {
	return projectionLoad{table: review.NewDueByDay(s.day)}
}

func (s ProjectionScheduler) GetShare(p projection.Settings, day time.Weekday) float64 {
	return convertSettingsToPreset(p).GetShare(day)
}

func (s ProjectionScheduler) IsNextCardDebt(a projection.Allowance, debt, begun int, isDue, isNew bool) bool {
	var day review.Allowance
	day.Backlog = a.Backlog
	return day.IsNextCardDebt(debt, begun, isDue, isNew)
}

// projectionLoad is the review count of cards due on each day.
type projectionLoad struct{ table *review.DueByDay }

func (l projectionLoad) Add(due time.Time) { l.table.Add(due) }

func (l projectionLoad) GetDayNumber(at time.Time) int { return l.table.GetDayNumber(at) }

func (l projectionLoad) PlaceCard(p projection.Settings, at, due time.Time) time.Time {
	return convertSettingsToPreset(p).AssignDueDay(l.table, at, due)
}

// ConvertPresetToSettings is a preset as the settings a projection runs on. The rule a card is learned by and
// its interval are the ones the preset counts by.
func ConvertPresetToSettings(p review.Preset) projection.Settings {
	rule, interval, _ := p.GetLearnedRule()
	learned := projection.RuleInterval
	if rule == review.RuleRetention {
		learned = projection.RuleRetention
	}
	return projection.Settings{
		Goal:                  convertGoalToProjection(p.Goal),
		By:                    p.By,
		MinutesADay:           p.MinutesADay,
		NewADay:               p.NewADay,
		ReviewsADay:           p.ReviewsADay,
		Retention:             p.Retention,
		Rule:                  learned,
		Interval:              interval,
		IsEveryShowingCharged: p.Counts == review.BudgetUnitShows,
		Backlog:               p.Backlog,
		Load:                  p.Load,
		IsEvenLoad:            p.IsEvenLoad,
	}
}

func convertSettingsToPreset(s projection.Settings) review.Preset {
	rule := review.RuleInterval
	if s.Rule == projection.RuleRetention {
		rule = review.RuleRetention
	}
	counts := review.BudgetUnitCards
	if s.IsEveryShowingCharged {
		counts = review.BudgetUnitShows
	}
	return review.Preset{
		Goal:        convertGoalToReview(s.Goal),
		By:          s.By,
		MinutesADay: s.MinutesADay,
		NewADay:     s.NewADay,
		ReviewsADay: s.ReviewsADay,
		Retention:   s.Retention,
		Rule:        rule,
		Interval:    s.Interval,
		Counts:      counts,
		Backlog:     s.Backlog,
		Load:        s.Load,
		IsEvenLoad:  s.IsEvenLoad,
	}
}

func ConvertSchedulesToCards(schedules map[review.CardID]review.Schedule) []projection.Card {
	out := make([]projection.Card, 0, len(schedules))
	for _, one := range schedules {
		out = append(out, convertScheduleToCard(one))
	}
	return out
}

func convertScheduleToCard(s review.Schedule) projection.Card {
	return projection.Card{
		Due: s.Due, Last: s.Last, Reps: s.Reps, Lapses: s.Lapses, Stability: s.Stability, Difficulty: s.Difficulty, Phase: s.Phase,
	}
}

func convertCardToSchedule(c projection.Card) review.Schedule {
	return review.Schedule{
		Due: c.Due, Last: c.Last, Reps: c.Reps, Lapses: c.Lapses, Stability: c.Stability, Difficulty: c.Difficulty, Phase: c.Phase,
	}
}

// ConvertCostToProjection is how long an answer takes, as a projection reads it.
func ConvertCostToProjection(c review.AnswerCost) projection.Cost {
	return projection.Cost{New: c.New, Review: c.Review}
}

// ConvertSpentToProjection is what a review day has used, as a projection reads it.
func ConvertSpentToProjection(s review.Spent) projection.Spent {
	return projection.Spent{Answered: s.Answered, New: s.New, Reviews: s.Reviews, Took: s.Took}
}

func convertBudgetName(name review.BudgetName) projection.BudgetName {
	switch name {
	case review.ClosedMinutes:
		return projection.ClosedMinutes
	case review.ClosedNew:
		return projection.ClosedNew
	case review.ClosedReviews:
		return projection.ClosedReviews
	case review.ClosedDate:
		return projection.ClosedDate
	case review.ClosedPaused:
		return projection.ClosedPaused
	case review.ClosedNothing, review.ClosedBacklog:
		return projection.ClosedNothing
	default:
		return projection.ClosedNothing
	}
}

func convertDaysToLearn(days int) int {
	if days == projection.NeverLearns {
		return review.NeverLearns
	}
	return days
}

func convertGoalToProjection(g review.Goal) projection.Goal {
	switch g {
	case review.GoalRetention:
		return projection.GoalRetention
	case review.GoalDate:
		return projection.GoalDate
	case review.GoalMinutes:
		return projection.GoalMinutes
	default:
		return projection.GoalMinutes
	}
}

func convertGoalToReview(g projection.Goal) review.Goal {
	switch g {
	case projection.GoalRetention:
		return review.GoalRetention
	case projection.GoalDate:
		return review.GoalDate
	case projection.GoalMinutes:
		return review.GoalMinutes
	default:
		return review.GoalMinutes
	}
}
