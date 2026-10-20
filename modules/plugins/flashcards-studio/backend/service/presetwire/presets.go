// Package presetwire carries a preset, its settings and its curve between the words the review speaks
// and the words the schema does.
//
// A preset is asked about from the editor's window and from the window a person
// runs their cards in, and the two hand the same settings and the same curve
// over.
package presetwire

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/review"
)

// GetSettingsBounds is how far each setting of a preset goes, as the schema
// carries it. They are the domain's own, so a client draws the field a person
// types into from what the write is held to.
func GetSettingsBounds() *v1.SettingsBounds {
	return &v1.SettingsBounds{
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
func getBounds(b review.Bounds) *v1.Bounds {
	return &v1.Bounds{Least: b.Least, Most: b.Most}
}

// ConvertPresetToWire is one preset as the schema carries it.
func ConvertPresetToWire(p flashcards.PresetContents, title string) *v1.Preset {
	return &v1.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: ConvertSettingsToWire(p.Settings),
		Problems: p.Problems,
		Stops:    ConvertStopReasonToWire(p.Stops),
		StopsOn:  ConvertStopReasonToWire(p.StopsToday),
	}
}

// ConvertStopReasonToWire is why a preset schedules nothing, as the schema names it.
func ConvertStopReasonToWire(s review.StopReason) v1.StopReason {
	switch s {
	case review.StoppedNoMinutes:
		return v1.StopReason_STOP_REASON_NO_MINUTES
	case review.StoppedNoCards:
		return v1.StopReason_STOP_REASON_NO_CARDS
	case review.StoppedNoDay:
		return v1.StopReason_STOP_REASON_NO_DAY
	case review.StoppedPastDay:
		return v1.StopReason_STOP_REASON_PAST_DAY
	case review.StoppedNoLoad:
		return v1.StopReason_STOP_REASON_NO_LOAD
	case review.StoppedNoWeek:
		return v1.StopReason_STOP_REASON_NO_WEEK
	case review.StoppedNothing:
		return v1.StopReason_STOP_REASON_NOTHING
	default:
		return v1.StopReason_STOP_REASON_UNSPECIFIED
	}
}

// ConvertSettingsToWire is how a preset schedules, as the schema carries it.
func ConvertSettingsToWire(p review.Preset) *v1.Settings {
	out := &v1.Settings{
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
func ConvertSettingsFromWire(s *v1.Settings) (review.Preset, error) {
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
	if written := strings.TrimSpace(s.GetByDate()); written != "" {
		day, err := time.Parse(review.DayLayout, written)
		if err != nil {
			return review.Preset{}, err
		}
		out.By = day
	}
	for _, name := range slices.Sorted(maps.Keys(s.GetLoad())) {
		day, known := review.Weekday(name)
		if !known {
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
func ConvertCurveToWire(c review.Curve) *v1.Curve {
	out := &v1.Curve{
		Goal:      ConvertGoalToWire(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*v1.Point, 0, len(c.Points)),
		Now:       convertCurvePoint(c.Now),
		Suggested: convertCurvePoint(c.Suggested),
		Decks:     int32(c.Decks),
		Cards:     int32(c.Cards),
		Overdue:   int32(c.Overdue),
		Unbegun:   int32(c.Unbegun),
	}
	for _, one := range c.Points {
		out.At = append(out.At, &v1.Point{
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
func ConvertBudgetsToWire(names review.BudgetNames) []v1.BudgetName {
	out := make([]v1.BudgetName, 0, len(names))
	for _, one := range names {
		out = append(out, ConvertBudgetToWire(one))
	}
	return out
}

// ConvertBudgetToWire is one budget, as the schema names it. Every budget the core holds
// is named here, so a budget added to the core is a case added beside them.
func ConvertBudgetToWire(name review.BudgetName) v1.BudgetName {
	switch name {
	case review.ClosedNothing:
		return v1.BudgetName_BUDGET_NAME_UNSPECIFIED
	case review.ClosedMinutes:
		return v1.BudgetName_BUDGET_NAME_MINUTES_A_DAY
	case review.ClosedNew:
		return v1.BudgetName_BUDGET_NAME_NEW_A_DAY
	case review.ClosedReviews:
		return v1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY
	case review.ClosedDate:
		return v1.BudgetName_BUDGET_NAME_BY_DATE
	case review.ClosedBacklog:
		return v1.BudgetName_BUDGET_NAME_BACKLOG
	case review.ClosedPaused:
		return v1.BudgetName_BUDGET_NAME_PAUSED
	}
	return v1.BudgetName_BUDGET_NAME_UNSPECIFIED
}

func convertCurvePoint(p review.CurvePoint) *v1.CurvePoint {
	return &v1.CurvePoint{At: int32(p.Index), Value: p.Value, Day: p.Day}
}

// ConvertGoalToWire is which value the control steers, as the schema names it.
func ConvertGoalToWire(g review.Goal) v1.Goal {
	switch g {
	case review.GoalMinutes:
		return v1.Goal_GOAL_MINUTES_A_DAY
	case review.GoalRetention:
		return v1.Goal_GOAL_RETENTION
	case review.GoalDate:
		return v1.Goal_GOAL_BY_DATE
	default:
		return v1.Goal_GOAL_UNSPECIFIED
	}
}

// ConvertGoalFromWire is the goal a client named, in the words the core holds it in. A goal
// the schema does not name is refused where the settings are weighed.
func ConvertGoalFromWire(g v1.Goal) review.Goal {
	switch g {
	case v1.Goal_GOAL_MINUTES_A_DAY:
		return review.GoalMinutes
	case v1.Goal_GOAL_RETENTION:
		return review.GoalRetention
	case v1.Goal_GOAL_BY_DATE:
		return review.GoalDate
	default:
		return ""
	}
}

// ConvertRuleToWire is what counts as learned, as the schema names it.
func ConvertRuleToWire(r review.LearnedRule) v1.Rule {
	switch r {
	case review.RuleInterval:
		return v1.Rule_RULE_INTERVAL
	case review.RuleRetention:
		return v1.Rule_RULE_RETENTION
	default:
		return v1.Rule_RULE_UNSPECIFIED
	}
}

// ConvertRuleFromWire is the rule a client named, in the words the core holds it in. A rule
// the schema does not name is refused where the settings are weighed.
func ConvertRuleFromWire(r v1.Rule) review.LearnedRule {
	switch r {
	case v1.Rule_RULE_INTERVAL:
		return review.RuleInterval
	case v1.Rule_RULE_RETENTION:
		return review.RuleRetention
	default:
		return ""
	}
}

// ConvertBudgetUnitToWire is what a day's budget is counted in, as the schema names it.
func ConvertBudgetUnitToWire(u review.BudgetUnit) v1.BudgetUnit {
	switch u {
	case review.BudgetUnitCards:
		return v1.BudgetUnit_BUDGET_UNIT_CARDS
	case review.BudgetUnitShows:
		return v1.BudgetUnit_BUDGET_UNIT_SHOWS
	default:
		return v1.BudgetUnit_BUDGET_UNIT_UNSPECIFIED
	}
}

// ConvertBudgetUnitFromWire is what a client said a day's budget is counted in, in the words
// the core holds it in. A value the schema does not name is refused where the
// settings are weighed.
func ConvertBudgetUnitFromWire(u v1.BudgetUnit) review.BudgetUnit {
	switch u {
	case v1.BudgetUnit_BUDGET_UNIT_CARDS:
		return review.BudgetUnitCards
	case v1.BudgetUnit_BUDGET_UNIT_SHOWS:
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
