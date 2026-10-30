package presetwire

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// GetSettingsBounds is how far each setting of a preset goes, as the schema
// carries it. They are the domain's own, so a client draws the field a person
func GetSettingsBounds() *flashcardsv1.SettingsBounds {
	return &flashcardsv1.SettingsBounds{
		MinutesADay: getBounds(review.MinutesADayBounds),
		NewADay:     getBounds(review.NewADayBounds),
		ReviewsADay: getBounds(review.ReviewsADayBounds),
		Retention:   getBounds(review.RetentionBounds),
		Backlog:     getBounds(review.BacklogBounds),
		Interval:    getBounds(review.IntervalBounds),
		Load:        getBounds(review.LoadBounds),
	}
}

// getBounds is one pair of bounds as the schema carries it.
func getBounds(b review.Bounds) *flashcardsv1.Bounds {
	return &flashcardsv1.Bounds{Min: b.Min, Max: b.Max}
}

// ConvertPresetToWire is one preset as the schema carries it.
func ConvertPresetToWire(p flashcards.PresetContents, title string) *flashcardsv1.Preset {
	return &flashcardsv1.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: ConvertSettingsToWire(p.Settings),
		Problems: p.Problems,
		Stops:    ConvertStopReasonToWire(p.Stops),
		StopsOn:  ConvertStopReasonToWire(p.StopsToday),
	}
}

// ConvertStopReasonToWire is why a preset schedules nothing, as the schema names it.
func ConvertStopReasonToWire(s review.StopReason) flashcardsv1.StopReason {
	switch s {
	case review.StoppedNoMinutes:
		return flashcardsv1.StopReason_STOP_REASON_NO_MINUTES
	case review.StoppedNoCards:
		return flashcardsv1.StopReason_STOP_REASON_NO_CARDS
	case review.StoppedNoDay:
		return flashcardsv1.StopReason_STOP_REASON_NO_DAY
	case review.StoppedPastDay:
		return flashcardsv1.StopReason_STOP_REASON_PAST_DAY
	case review.StoppedNoLoad:
		return flashcardsv1.StopReason_STOP_REASON_NO_LOAD
	case review.StoppedNoWeek:
		return flashcardsv1.StopReason_STOP_REASON_NO_WEEK
	case review.StoppedNothing:
		return flashcardsv1.StopReason_STOP_REASON_NOTHING
	default:
		return flashcardsv1.StopReason_STOP_REASON_UNSPECIFIED
	}
}

// ConvertSettingsToWire is how a preset schedules, as the schema carries it.
func ConvertSettingsToWire(p review.Preset) *flashcardsv1.Settings {
	out := &flashcardsv1.Settings{
		Goal:        ConvertGoalToWire(p.Goal),
		Counts:      ConvertBudgetUnitToWire(p.Counts),
		MinutesADay: int32(p.MinutesADay),
		NewADay:     int32(p.NewADay),
		ReviewsADay: int32(p.ReviewsADay),
		Retention:   p.Retention,
		Learned:     ConvertRuleToWire(p.Rule),
		Interval:    int32(p.Interval),
		Backlog:     int32(p.Backlog),
		HasEvenLoad: p.IsEvenLoad,
		Load:        make(map[string]int32, len(p.Load)),
	}
	if !p.By.IsZero() {
		out.ByDate = p.By.Format(review.DayLayout)
	}
	for day, share := range p.Load {
		out.Load[review.DayName(day)] = int32(share)
	}
	return out
}

// ConvertSettingsFromWire is the settings a client is putting into a preset, in the words
// the core holds them in. A day that is not one and a load kept on something
// that is not a day of the week are the client's to correct.
func ConvertSettingsFromWire(s *flashcardsv1.Settings) (review.Preset, error) {
	out := review.Preset{
		Goal:        ConvertGoalFromWire(s.GetGoal()),
		Counts:      ConvertBudgetUnitFromWire(s.GetCounts()),
		MinutesADay: int(s.GetMinutesADay()),
		NewADay:     int(s.GetNewADay()),
		ReviewsADay: int(s.GetReviewsADay()),
		Retention:   s.GetRetention(),
		Rule:        ConvertRuleFromWire(s.GetLearned()),
		Interval:    int(s.GetInterval()),
		Backlog:     int(s.GetBacklog()),
		IsEvenLoad:  s.GetHasEvenLoad(),
	}
	if content := strings.TrimSpace(s.GetByDate()); content != "" {
		day, err := time.Parse(review.DayLayout, content)
		if err != nil {
			return review.Preset{}, err
		}
		out.By = day
	}
	for _, name := range slices.Sorted(maps.Keys(s.GetLoad())) {
		day, recognized := review.Weekday(name)
		if !recognized {
			return review.Preset{}, errors.New(name + " is not a day of the week")
		}
		if out.Load == nil {
			out.Load = make(map[time.Weekday]int, len(s.GetLoad()))
		}
		out.Load[day] = int(s.GetLoad()[name])
	}
	return out, nil
}

// ConvertCurveToWire is a curve as the schema carries it.
func ConvertCurveToWire(c flashcards.Curve) *flashcardsv1.Curve {
	out := &flashcardsv1.Curve{
		Goal:      ConvertGoalToWire(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*flashcardsv1.Point, 0, len(c.Points)),
		Now:       convertCurvePoint(c.Now),
		Suggested: convertCurvePoint(c.Suggested),
		Decks:     int32(c.Decks),
		Cards:     int32(c.Cards),
		Overdue:   int32(c.Overdue),
		Unbegun:   int32(c.Unbegun),
	}
	for _, one := range c.Points {
		out.At = append(out.At, &flashcardsv1.Point{
			Reviews:  one.Reviews,
			Minutes:  one.Minutes,
			Retained: one.Retained,
			Owed:     int32(one.Owed),
			Through:  one.Share,
			IsEnough: one.IsEnough,
			Closed:   ConvertBudgetsToWire(one.Closed),
			Short:    int32(one.Short),
			Clears:   int32(one.Clears),
			Learned:  int32(one.Learned),
			Learns:   convertLearnedDay(one.Learns),
			Backlog:  convertBacklog(one.Backlog),
		})
	}
	return out
}

// ConvertBudgetsToWire are the budgets that closed a day, as the schema names them.
func ConvertBudgetsToWire(names review.BudgetNames) []flashcardsv1.BudgetName {
	out := make([]flashcardsv1.BudgetName, 0, len(names))
	for _, one := range names {
		out = append(out, ConvertBudgetToWire(one))
	}
	return out
}

// ConvertBudgetToWire is one budget, as the schema names it. Every budget the core holds
// is named here, so a budget added to the core is a case added beside them.
func ConvertBudgetToWire(name review.BudgetName) flashcardsv1.BudgetName {
	switch name {
	case review.ClosedNothing:
		return flashcardsv1.BudgetName_BUDGET_NAME_UNSPECIFIED
	case review.ClosedMinutes:
		return flashcardsv1.BudgetName_BUDGET_NAME_MINUTES_A_DAY
	case review.ClosedNew:
		return flashcardsv1.BudgetName_BUDGET_NAME_NEW_A_DAY
	case review.ClosedReviews:
		return flashcardsv1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY
	case review.ClosedDate:
		return flashcardsv1.BudgetName_BUDGET_NAME_BY_DATE
	case review.ClosedBacklog:
		return flashcardsv1.BudgetName_BUDGET_NAME_BACKLOG
	case review.ClosedPaused:
		return flashcardsv1.BudgetName_BUDGET_NAME_PAUSED
	}
	return flashcardsv1.BudgetName_BUDGET_NAME_UNSPECIFIED
}

func convertCurvePoint(p flashcards.CurvePoint) *flashcardsv1.CurvePoint {
	return &flashcardsv1.CurvePoint{At: int32(p.Index), Value: p.Value, Day: p.Day}
}

// ConvertGoalToWire is which value the control steers, as the schema names it.
func ConvertGoalToWire(g review.Goal) flashcardsv1.Goal {
	switch g {
	case review.GoalMinutes:
		return flashcardsv1.Goal_GOAL_MINUTES_A_DAY
	case review.GoalRetention:
		return flashcardsv1.Goal_GOAL_RETENTION
	case review.GoalDate:
		return flashcardsv1.Goal_GOAL_BY_DATE
	default:
		return flashcardsv1.Goal_GOAL_UNSPECIFIED
	}
}

func ConvertGoalFromWire(g flashcardsv1.Goal) review.Goal {
	switch g {
	case flashcardsv1.Goal_GOAL_MINUTES_A_DAY:
		return review.GoalMinutes
	case flashcardsv1.Goal_GOAL_RETENTION:
		return review.GoalRetention
	case flashcardsv1.Goal_GOAL_BY_DATE:
		return review.GoalDate
	default:
		return ""
	}
}

// ConvertRuleToWire is what counts as learned, as the schema names it.
func ConvertRuleToWire(r review.LearnedRule) flashcardsv1.Rule {
	switch r {
	case review.RuleInterval:
		return flashcardsv1.Rule_RULE_INTERVAL
	case review.RuleRetention:
		return flashcardsv1.Rule_RULE_RETENTION
	default:
		return flashcardsv1.Rule_RULE_UNSPECIFIED
	}
}

func ConvertRuleFromWire(r flashcardsv1.Rule) review.LearnedRule {
	switch r {
	case flashcardsv1.Rule_RULE_INTERVAL:
		return review.RuleInterval
	case flashcardsv1.Rule_RULE_RETENTION:
		return review.RuleRetention
	default:
		return ""
	}
}

// ConvertBudgetUnitToWire is what a day's budget is counted in, as the schema names it.
func ConvertBudgetUnitToWire(u review.BudgetUnit) flashcardsv1.BudgetUnit {
	switch u {
	case review.BudgetUnitCards:
		return flashcardsv1.BudgetUnit_BUDGET_UNIT_CARDS
	case review.BudgetUnitShows:
		return flashcardsv1.BudgetUnit_BUDGET_UNIT_SHOWS
	default:
		return flashcardsv1.BudgetUnit_BUDGET_UNIT_UNSPECIFIED
	}
}

// ConvertBudgetUnitFromWire is what a client said a day's budget is counted in, in the words
// settings are weighed.
func ConvertBudgetUnitFromWire(u flashcardsv1.BudgetUnit) review.BudgetUnit {
	switch u {
	case flashcardsv1.BudgetUnit_BUDGET_UNIT_CARDS:
		return review.BudgetUnitCards
	case flashcardsv1.BudgetUnit_BUDGET_UNIT_SHOWS:
		return review.BudgetUnitShows
	default:
		return ""
	}
}

// convertLearnedDay is the day the whole material stands learned, as the schema
// carries it. A place with no such day to name carries none.
func convertLearnedDay(day int) *int32 {
	if day == review.LearnsUnasked {
		return nil
	}
	out := int32(day)
	return &out
}

// convertBacklog is a convertBacklog day by day, as the schema carries one.
func convertBacklog(days []int) []int32 {
	out := make([]int32, 0, len(days))
	for _, one := range days {
		out = append(out, int32(one))
	}
	return out
}
