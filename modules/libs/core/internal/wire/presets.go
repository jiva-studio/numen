// Package wire carries what more than one adapter puts on the schema.
//
// A preset is asked about from the editor's window and from the window a person
// runs their cards in, and the two hand the same settings and the same curve
// over.
package wire

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// SettingsBounds is how far each setting of a preset goes, as the schema
// carries it. They are the domain's own, so a client draws the field a person
// types into from what the write is held to.
func SettingsBounds() *v1.SettingsBounds {
	return &v1.SettingsBounds{
		MinutesADay: bounded(review.MinutesADayBounds),
		NewADay:     bounded(review.NewADayBounds),
		ReviewsADay: bounded(review.ReviewsADayBounds),
		Retention:   bounded(review.RetentionBounds),
		Backlog:     bounded(review.BacklogBounds),
		Interval:    bounded(review.IntervalBounds),
		Load:        bounded(review.LoadBounds),
	}
}

// bounded is one pair of bounds as the schema carries it.
func bounded(b review.Bounds) *v1.Bounds {
	return &v1.Bounds{Least: b.Least, Most: b.Most}
}

// PresetOf is one preset as the schema carries it.
func PresetOf(p flashcards.PresetContents, title string) *v1.Preset {
	return &v1.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: SettingsOf(p.Settings),
		Problems: p.Problems,
		Stops:    StopReasonOf(p.Stops),
		StopsOn:  StopReasonOf(p.StopsToday),
	}
}

// StopReasonOf is why a preset schedules nothing, as the schema names it.
func StopReasonOf(s review.StopReason) v1.StopReason {
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

// SettingsOf is how a preset schedules, as the schema carries it.
func SettingsOf(p review.Preset) *v1.Settings {
	out := &v1.Settings{
		Goal:        GoalOf(p.Goal),
		Counts:      CountsOf(p.Counts),
		MinutesADay: int32(p.MinutesADay),
		NewADay:     int32(p.NewADay),
		ReviewsADay: int32(p.ReviewsADay),
		Retention:   p.Retention,
		Learned:     RuleOf(p.Rule),
		Interval:    int32(p.Interval),
		Backlog:     int32(p.Backlog),
		EvenLoad:    p.EvenLoad,
		Load:        make(map[string]int32, len(p.Load)),
	}
	if !p.By.IsZero() {
		out.ByDate = p.By.Format(review.Named)
	}
	for day, share := range p.Load {
		out.Load[review.DayName(day)] = int32(share)
	}
	return out
}

// SettingsIn is the settings a client is putting into a preset, in the words
// the core holds them in. A day that is not one and a load kept on something
// that is not a day of the week are the client's to correct.
func SettingsIn(s *v1.Settings) (review.Preset, error) {
	out := review.Preset{
		Goal:        GoalIn(s.GetGoal()),
		Counts:      CountsIn(s.GetCounts()),
		MinutesADay: int(s.GetMinutesADay()),
		NewADay:     int(s.GetNewADay()),
		ReviewsADay: int(s.GetReviewsADay()),
		Retention:   s.GetRetention(),
		Rule:        RuleIn(s.GetLearned()),
		Interval:    int(s.GetInterval()),
		Backlog:     int(s.GetBacklog()),
		EvenLoad:    s.GetEvenLoad(),
	}
	if written := strings.TrimSpace(s.GetByDate()); written != "" {
		day, err := time.Parse(review.Named, written)
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

// CurveOf is a curve as the schema carries it.
func CurveOf(c review.Curve) *v1.Curve {
	out := &v1.Curve{
		Goal:      GoalOf(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*v1.Point, 0, len(c.Points)),
		Now:       placeOf(c.Now),
		Suggested: placeOf(c.Suggested),
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
			Enough:   one.Enough,
			Closed:   one.Closed.Names(),
			Short:    int32(one.Short),
			Clears:   int32(one.Clears),
			Learned:  int32(one.Learned),
			Learns:   learns(one.Learns),
			Backlog:  backlog(one.Backlog),
		})
	}
	return out
}

func placeOf(p review.Place) *v1.Place {
	return &v1.Place{At: int32(p.Index), Value: p.Value, Day: p.Day}
}

// GoalOf is which value the control steers, as the schema names it.
func GoalOf(g review.Goal) v1.Goal {
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

// GoalIn is the goal a client named, in the words the core holds it in. A goal
// the schema does not name is refused where the settings are weighed.
func GoalIn(g v1.Goal) review.Goal {
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

// RuleOf is what counts as learned, as the schema names it.
func RuleOf(r review.LearnedRule) v1.Rule {
	switch r {
	case review.RuleInterval:
		return v1.Rule_RULE_INTERVAL
	case review.RuleRetention:
		return v1.Rule_RULE_RETENTION
	default:
		return v1.Rule_RULE_UNSPECIFIED
	}
}

// RuleIn is the rule a client named, in the words the core holds it in. A rule
// the schema does not name is refused where the settings are weighed.
func RuleIn(r v1.Rule) review.LearnedRule {
	switch r {
	case v1.Rule_RULE_INTERVAL:
		return review.RuleInterval
	case v1.Rule_RULE_RETENTION:
		return review.RuleRetention
	default:
		return ""
	}
}

// CountsOf is what a day's budget is spent on, as the schema names it.
func CountsOf(c review.Counts) v1.Counts {
	switch c {
	case review.CountsCards:
		return v1.Counts_COUNTS_CARDS
	case review.CountsShows:
		return v1.Counts_COUNTS_SHOWS
	default:
		return v1.Counts_COUNTS_UNSPECIFIED
	}
}

// CountsIn is what a client said a day's budget is spent on, in the words the
// core holds it in. A value the schema does not name is refused where the
// settings are weighed.
func CountsIn(c v1.Counts) review.Counts {
	switch c {
	case v1.Counts_COUNTS_CARDS:
		return review.CountsCards
	case v1.Counts_COUNTS_SHOWS:
		return review.CountsShows
	default:
		return ""
	}
}

// learns is the day the whole material stands learned, as the schema carries
// it. A place with no such day to name carries none.
func learns(day int) *int32 {
	if day == review.LearnsUnasked {
		return nil
	}
	out := int32(day)
	return &out
}

// backlog is a backlog day by day, as the schema carries one.
func backlog(days []int) []int32 {
	out := make([]int32, 0, len(days))
	for _, one := range days {
		out = append(out, int32(one))
	}
	return out
}
