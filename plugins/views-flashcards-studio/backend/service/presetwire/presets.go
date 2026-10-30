// Package presetwire converts presets between the domain types and the schema:
// the numbers are the domain's, and the words are the schema's.
//
// A preset is queried from the editor window and from the review window, and both
// receive the same settings and the same curve.
package presetwire

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// GetSettingsBounds returns how far each setting of a preset may go, as the
// schema carries it. The bounds are the domain's own, so a client draws each field
// with the limits the domain enforces.
func GetSettingsBounds() *api.SettingsBounds {
	return &api.SettingsBounds{
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
func getBounds(b review.Bounds) *api.Bounds {
	return &api.Bounds{Min: b.Min, Max: b.Max}
}

// ConvertPresetToWire is one preset as the schema carries it.
func ConvertPresetToWire(p flashcards.PresetContents, title string) *api.Preset {
	return &api.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: ConvertSettingsToWire(p.Settings),
		Problems: p.Problems,
		Stops:    ConvertStopReasonToWire(p.Stops),
		StopsOn:  ConvertStopReasonToWire(p.StopsToday),
	}
}

// ConvertStopReasonToWire is why a preset schedules nothing, as the schema names it.
func ConvertStopReasonToWire(s review.StopReason) api.StopReason {
	switch s {
	case review.StoppedNoMinutes:
		return api.StopReasonNoMinutes
	case review.StoppedNoCards:
		return api.StopReasonNoCards
	case review.StoppedNoDay:
		return api.StopReasonNoDay
	case review.StoppedPastDay:
		return api.StopReasonPastDay
	case review.StoppedNoLoad:
		return api.StopReasonNoLoad
	case review.StoppedNoWeek:
		return api.StopReasonNoWeek
	case review.StoppedNothing:
		return api.StopReasonNothing
	default:
		return api.StopReasonUnspecified
	}
}

// ConvertSettingsToWire is how a preset schedules, as the schema carries it.
func ConvertSettingsToWire(p review.Preset) *api.Settings {
	out := &api.Settings{
		Goal:        ConvertGoalToWire(p.Goal),
		ByDate:      "",
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
func ConvertSettingsFromWire(s *api.Settings) (review.Preset, error) {
	if s == nil {
		s = &api.Settings{
			Goal: api.GoalUnspecified, ByDate: "", MinutesADay: 0, NewADay: 0, ReviewsADay: 0, Retention: 0,
			HasEvenLoad: false, Counts: api.BudgetUnitUnspecified, Backlog: 0, Load: nil,
			Learned: api.RuleUnspecified, Interval: 0,
		}
	}
	out := review.Preset{
		Goal:        ConvertGoalFromWire(s.Goal),
		By:          time.Time{},
		Load:        nil,
		Counts:      ConvertBudgetUnitFromWire(s.Counts),
		MinutesADay: int(s.MinutesADay),
		NewADay:     int(s.NewADay),
		ReviewsADay: int(s.ReviewsADay),
		Retention:   s.Retention,
		Rule:        ConvertRuleFromWire(s.Learned),
		Interval:    int(s.Interval),
		Backlog:     int(s.Backlog),
		IsEvenLoad:  s.HasEvenLoad,
	}
	if content := strings.TrimSpace(s.ByDate); content != "" {
		day, err := time.Parse(review.DayLayout, content)
		if err != nil {
			return review.Preset{}, err
		}
		out.By = day
	}
	for _, name := range slices.Sorted(maps.Keys(s.Load)) {
		day, recognized := review.Weekday(name)
		if !recognized {
			return review.Preset{}, errors.New(name + " is not a day of the week")
		}
		if out.Load == nil {
			out.Load = make(map[time.Weekday]int, len(s.Load))
		}
		out.Load[day] = int(s.Load[name])
	}
	return out, nil
}

// ConvertCurveToWire is a curve as the schema carries it.
func ConvertCurveToWire(c flashcards.Curve) *api.Curve {
	out := &api.Curve{
		Goal:      ConvertGoalToWire(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*api.Point, 0, len(c.Points)),
		Now:       convertCurvePoint(c.Now),
		Suggested: convertCurvePoint(c.Suggested),
		Decks:     int32(c.Decks),
		Cards:     int32(c.Cards),
		Overdue:   int32(c.Overdue),
		Unbegun:   int32(c.Unbegun),
	}
	for _, one := range c.Points {
		out.At = append(out.At, &api.Point{
			Reviews:  one.Reviews,
			Minutes:  one.Minutes,
			Retained: one.Retained,
			Due:      int32(one.Due),
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
func ConvertBudgetsToWire(names review.BudgetNames) []api.BudgetName {
	out := make([]api.BudgetName, 0, len(names))
	for _, one := range names {
		out = append(out, ConvertBudgetToWire(one))
	}
	return out
}

// ConvertBudgetToWire is one budget, as the schema names it. Every budget the core holds
// is named here, so a budget added to the core is a case added beside them.
func ConvertBudgetToWire(name review.BudgetName) api.BudgetName {
	switch name {
	case review.ClosedNothing:
		return api.BudgetNameUnspecified
	case review.ClosedMinutes:
		return api.BudgetNameMinutesADay
	case review.ClosedNew:
		return api.BudgetNameNewADay
	case review.ClosedReviews:
		return api.BudgetNameReviewsADay
	case review.ClosedDate:
		return api.BudgetNameByDate
	case review.ClosedBacklog:
		return api.BudgetNameBacklog
	case review.ClosedPaused:
		return api.BudgetNamePaused
	}
	return api.BudgetNameUnspecified
}

func convertCurvePoint(p flashcards.CurvePoint) *api.CurvePoint {
	return &api.CurvePoint{At: int32(p.Index), Value: p.Value, Day: p.Day}
}

// ConvertGoalToWire is which value the control steers, as the schema names it.
func ConvertGoalToWire(g review.Goal) api.Goal {
	switch g {
	case review.GoalMinutes:
		return api.GoalMinutesADay
	case review.GoalRetention:
		return api.GoalRetention
	case review.GoalDate:
		return api.GoalByDate
	default:
		return api.GoalUnspecified
	}
}

// ConvertGoalFromWire converts the goal a client sent into the domain goal.
func ConvertGoalFromWire(g api.Goal) review.Goal {
	switch g {
	case api.GoalMinutesADay:
		return review.GoalMinutes
	case api.GoalRetention:
		return review.GoalRetention
	case api.GoalByDate:
		return review.GoalDate
	case api.GoalUnspecified:
		return ""
	default:
		return ""
	}
}

// ConvertRuleToWire is what counts as learned, as the schema names it.
func ConvertRuleToWire(r review.LearnedRule) api.Rule {
	switch r {
	case review.RuleInterval:
		return api.RuleInterval
	case review.RuleRetention:
		return api.RuleRetention
	default:
		return api.RuleUnspecified
	}
}

// ConvertRuleFromWire converts the learned rule a client sent into the domain rule.
func ConvertRuleFromWire(r api.Rule) review.LearnedRule {
	switch r {
	case api.RuleInterval:
		return review.RuleInterval
	case api.RuleRetention:
		return review.RuleRetention
	case api.RuleUnspecified:
		return ""
	default:
		return ""
	}
}

// ConvertBudgetUnitToWire is what a day's budget is counted in, as the schema names it.
func ConvertBudgetUnitToWire(u review.BudgetUnit) api.BudgetUnit {
	switch u {
	case review.BudgetUnitCards:
		return api.BudgetUnitCards
	case review.BudgetUnitShows:
		return api.BudgetUnitShows
	default:
		return api.BudgetUnitUnspecified
	}
}

// ConvertBudgetUnitFromWire converts the unit that a client sent for a day's
// budget into the domain's budget unit.
func ConvertBudgetUnitFromWire(u api.BudgetUnit) review.BudgetUnit {
	switch u {
	case api.BudgetUnitCards:
		return review.BudgetUnitCards
	case api.BudgetUnitShows:
		return review.BudgetUnitShows
	case api.BudgetUnitUnspecified:
		return ""
	default:
		return ""
	}
}

// convertLearnedDay returns the day on which the whole material is learned, as
// the schema carries it. A projection with no such day returns nil.
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
