// Package wire carries what more than one adapter puts on the schema.
//
// A preset is asked about from the editor's window and from the window a person
// runs their cards in, and the two hand the same settings and the same curve
// over.
package wire

import (
	"errors"
	"strings"
	"time"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// PresetOf is one preset as the schema carries it.
func PresetOf(p flashcards.Preset, title string) *v1.Preset {
	return &v1.Preset{
		Path:     p.Path,
		Title:    title,
		Settings: SettingsOf(p.Preset),
		Problems: p.Problems,
	}
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
		EvenLoad:    p.EvenLoad,
		LightDays:   make([]string, 0, len(p.LightDays)),
	}
	if !p.By.IsZero() {
		out.ByDate = p.By.Format(history.Named)
	}
	for _, day := range p.LightDays {
		out.LightDays = append(out.LightDays, history.DayName(day))
	}
	return out
}

// SettingsIn is the settings a client is putting into a preset, in the words
// the core holds them in. A day that is not one and a light day that is not a
// day of the week are the client's to correct.
func SettingsIn(s *v1.Settings) (history.Preset, error) {
	out := history.Preset{
		Goal:        GoalIn(s.GetGoal()),
		Counts:      CountsIn(s.GetCounts()),
		MinutesADay: int(s.GetMinutesADay()),
		NewADay:     int(s.GetNewADay()),
		ReviewsADay: int(s.GetReviewsADay()),
		Retention:   s.GetRetention(),
		EvenLoad:    s.GetEvenLoad(),
	}
	if written := strings.TrimSpace(s.GetByDate()); written != "" {
		day, err := time.Parse(history.Named, written)
		if err != nil {
			return history.Preset{}, err
		}
		out.By = day
	}
	for _, name := range s.GetLightDays() {
		day, known := history.Weekday(name)
		if !known {
			return history.Preset{}, errors.New(name + " is not a day of the week")
		}
		out.LightDays = append(out.LightDays, day)
	}
	return out, nil
}

// CurveOf is a curve as the schema carries it.
func CurveOf(c flashcards.Curve) *v1.Curve {
	out := &v1.Curve{
		Goal:      GoalOf(c.Goal),
		Grid:      c.Grid,
		Days:      c.Days,
		At:        make([]*v1.Point, 0, len(c.At)),
		Now:       markOf(c.Now),
		Suggested: markOf(c.Suggested),
		Decks:     int32(c.Decks),
		Cards:     int32(c.Cards),
		Overdue:   int32(c.Overdue),
	}
	for _, one := range c.At {
		out.At = append(out.At, &v1.Point{
			Reviews:  one.Reviews,
			Minutes:  one.Minutes,
			Retained: one.Retained,
			Owed:     int32(one.Owed),
			Through:  one.Through,
			Enough:   one.Enough,
			Met:      one.Met,
			Closed:   string(one.Closed),
			Clears:   int32(one.Clears),
		})
	}
	return out
}

func markOf(m flashcards.Mark) *v1.Mark {
	return &v1.Mark{At: int32(m.At), Value: m.Value, Day: m.Day}
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
