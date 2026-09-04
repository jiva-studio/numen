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

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// PresetOf is one preset as the schema carries it.
func PresetOf(p flashcards.PresetContents, title string) *v1.Preset {
	return &v1.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: SettingsOf(p.Settings),
		Problems: p.Problems,
		Stops:    StoppedOf(p.Stops),
		StopsOn:  StoppedOf(p.StopsToday),
	}
}

// StoppedOf is why a preset schedules nothing, as the schema names it.
func StoppedOf(s history.StopReason) v1.Stopped {
	switch s {
	case history.StoppedNoMinutes:
		return v1.Stopped_STOPPED_NO_MINUTES
	case history.StoppedNoCards:
		return v1.Stopped_STOPPED_NO_CARDS
	case history.StoppedNoDay:
		return v1.Stopped_STOPPED_NO_DAY
	case history.StoppedPastDay:
		return v1.Stopped_STOPPED_PAST_DAY
	case history.StoppedNoLoad:
		return v1.Stopped_STOPPED_NO_LOAD
	case history.StoppedNoWeek:
		return v1.Stopped_STOPPED_NO_WEEK
	}
	return v1.Stopped_STOPPED_NOTHING
}

// SettingsOf is how a preset schedules, as the schema carries it.
func SettingsOf(p history.Preset) *v1.Settings {
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
		out.ByDate = p.By.Format(history.Named)
	}
	for day, share := range p.Load {
		out.Load[history.DayName(day)] = int32(share)
	}
	return out
}

// SettingsIn is the settings a client is putting into a preset, in the words
// the core holds them in. A day that is not one and a load kept on something
// that is not a day of the week are the client's to correct.
func SettingsIn(s *v1.Settings) (history.Preset, error) {
	out := history.Preset{
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
		day, err := time.Parse(history.Named, written)
		if err != nil {
			return history.Preset{}, err
		}
		out.By = day
	}
	for _, name := range slices.Sorted(maps.Keys(s.GetLoad())) {
		day, known := history.Weekday(name)
		if !known {
			return history.Preset{}, errors.New(name + " is not a day of the week")
		}
		if out.Load == nil {
			out.Load = make(map[time.Weekday]int, len(s.GetLoad()))
		}
		out.Load[day] = int(s.GetLoad()[name])
	}
	return out, nil
}

// CurveOf is a curve as the schema carries it.
func CurveOf(c flashcards.Curve) *v1.Curve {
	out := &v1.Curve{
		Goal:      GoalOf(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*v1.Point, 0, len(c.Points)),
		Now:       markOf(c.Now),
		Suggested: markOf(c.Suggested),
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

func markOf(m flashcards.Place) *v1.Mark {
	return &v1.Mark{At: int32(m.Index), Value: m.Value, Day: m.Day}
}

// GoalOf is which value the control steers, as the schema names it.
func GoalOf(g history.Goal) v1.Goal {
	switch g {
	case history.GoalMinutes:
		return v1.Goal_GOAL_MINUTES_A_DAY
	case history.GoalRetention:
		return v1.Goal_GOAL_RETENTION
	case history.GoalDate:
		return v1.Goal_GOAL_BY_DATE
	default:
		return v1.Goal_GOAL_UNSPECIFIED
	}
}

// GoalIn is the goal a client named, in the words the core holds it in. A goal
// the schema does not name is refused where the settings are weighed.
func GoalIn(g v1.Goal) history.Goal {
	switch g {
	case v1.Goal_GOAL_MINUTES_A_DAY:
		return history.GoalMinutes
	case v1.Goal_GOAL_RETENTION:
		return history.GoalRetention
	case v1.Goal_GOAL_BY_DATE:
		return history.GoalDate
	default:
		return ""
	}
}

// RuleOf is what counts as learned, as the schema names it.
func RuleOf(r history.LearnedRule) v1.Rule {
	switch r {
	case history.RuleInterval:
		return v1.Rule_RULE_INTERVAL
	case history.RuleRetention:
		return v1.Rule_RULE_RETENTION
	default:
		return v1.Rule_RULE_UNSPECIFIED
	}
}

// RuleIn is the rule a client named, in the words the core holds it in. A rule
// the schema does not name is refused where the settings are weighed.
func RuleIn(r v1.Rule) history.LearnedRule {
	switch r {
	case v1.Rule_RULE_INTERVAL:
		return history.RuleInterval
	case v1.Rule_RULE_RETENTION:
		return history.RuleRetention
	default:
		return ""
	}
}

// CountsOf is what a day's budget is spent on, as the schema names it.
func CountsOf(c history.Counts) v1.Counts {
	switch c {
	case history.CountsCards:
		return v1.Counts_COUNTS_CARDS
	case history.CountsShows:
		return v1.Counts_COUNTS_SHOWS
	default:
		return v1.Counts_COUNTS_UNSPECIFIED
	}
}

// CountsIn is what a client said a day's budget is spent on, in the words the
// core holds it in. A value the schema does not name is refused where the
// settings are weighed.
func CountsIn(c v1.Counts) history.Counts {
	switch c {
	case v1.Counts_COUNTS_CARDS:
		return history.CountsCards
	case v1.Counts_COUNTS_SHOWS:
		return history.CountsShows
	default:
		return ""
	}
}

// learns is the day the whole material stands learned, as the schema carries
// it. A place with no such day to name carries none.
func learns(day int) *int32 {
	if day == history.LearnsUnasked {
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
