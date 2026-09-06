package review_test

import (
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// lastAnswer is the instant the schedules below were last answered at.
var lastAnswer = time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)

// whereItStands generates where the answers so far have left one card face: how
// long it is sent away for, and how long the scheduler expects it to stay
// recalled. A face nobody has answered is drawn as often as one that has been,
// and carries a day it is wanted on like any other: what says nobody has
// answered it is that it was never answered, and not that nothing is set on it.
func whereItStands(t *rapid.T) review.Schedule {
	// Whole days as often as the minutes between them: an interval is counted in
	// days, so the day it falls exactly on is the case that decides the rule.
	away := time.Duration(rapid.IntRange(0, 400).Draw(t, "away"))*24*time.Hour +
		time.Duration(rapid.SampledFrom([]int{0, 0, 0, 1, 59, 1439}).Draw(t, "minutes"))*time.Minute
	out := review.Schedule{
		Due:       lastAnswer.Add(away),
		Stability: float64(rapid.IntRange(0, 500).Draw(t, "stability")),
	}
	if rapid.Bool().Draw(t, "seen") {
		out.Last = lastAnswer
	}
	return out
}

// ruleNamed generates a preset's three settings for the rule, over the whole of
// what each may be and past both ends of it: a value a rule cannot hold is what
// a preset written by hand carries.
func ruleNamed(t *rapid.T) review.Preset {
	return review.Preset{
		Rule: review.LearnedRule(rapid.SampledFrom([]string{
			string(review.RuleInterval), string(review.RuleRetention), "", "sometimes",
		}).Draw(t, "rule")),
		Interval:  rapid.IntRange(-5, 400).Draw(t, "interval"),
		Retention: float64(rapid.IntRange(0, 100).Draw(t, "retention")) / 100,
	}
}

// A card face nobody has answered is learned by neither rule, whatever the
// preset says and whenever it is asked.
//
// A card face nobody has answered is learned by neither rule.
func TestACardFaceNobodyHasAnsweredIsLearnedByNeitherRule(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		p := ruleNamed(t)
		s := whereItStands(t)
		s.Last = time.Time{}
		at := lastAnswer.Add(time.Duration(rapid.IntRange(-1000, 1000).Draw(t, "days")) * 24 * time.Hour)
		if p.Learned(s, at) {
			t.Fatalf("a card face nobody has answered counts learned under %+v "+
				"at %v, standing at %+v", p, at, s)
		}
	})
}

// A preset naming a rule it does not have, or a value the rule cannot hold,
// counts by the default rule at its default value. A key nobody wrote leaves
// the default in force, and never a threshold every card passes.
//
// A preset that names no rule counts by the default rule at its default value.
func TestAValueTheRuleCannotHoldCountsByTheDefault(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		p := ruleNamed(t)
		s := whereItStands(t)
		at := lastAnswer.Add(time.Duration(rapid.IntRange(0, 1000).Draw(t, "days")) * 24 * time.Hour)

		standard := review.Defaults()
		if review.KnownRule(p.Rule) {
			standard.Rule = p.Rule
		}
		if review.IntervalBounds.Holds(float64(p.Interval)) {
			standard.Interval = p.Interval
		}
		if review.RetentionBounds.Holds(p.Retention) {
			standard.Retention = p.Retention
		}
		if got, want := p.Learned(s, at), standard.Learned(s, at); got != want {
			t.Fatalf("%+v counts %+v learned as %v at %v, where the rule in "+
				"force is %+v and counts it %v", p, s, got, at, standard, want)
		}
	})
}

// Under a rule of an interval a card face is learned once it is sent away for
// the preset's interval or longer, so the further away it is sent the more
// presets count it learned: a card learned under one interval is learned under
// every shorter one.
//
// Under learned: interval a card is learned once the interval it is sent away
// for reaches the preset's interval.
func TestALongerIntervalIsLearnedByEveryShorterOne(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		s := whereItStands(t)
		short := rapid.IntRange(1, 365).Draw(t, "shorter")
		long := rapid.IntRange(short, 365).Draw(t, "longer")
		at := lastAnswer.Add(time.Duration(rapid.IntRange(0, 1000).Draw(t, "days")) * 24 * time.Hour)

		by := func(days int) review.Preset {
			return review.Preset{Rule: review.RuleInterval, Interval: days}
		}
		if by(long).Learned(s, at) && !by(short).Learned(s, at) {
			t.Fatalf("a card face sent away %v counts learned at an interval of "+
				"%d days and not at one of %d", s.Due.Sub(s.Last), long, short)
		}
	})
}

// Under a rule of retention, learned is where the card face stands now and not
// a milestone it passed: the chance of recalling it only falls as the days go
// by, so a card counted learned on a later day was counted learned on every
// earlier one, and a lapse that collapses its stability takes it out of that
// standing.
//
// Learned is a state and not a milestone, asked of where the card stands now.
func TestRecallIsAStateAndNotAMilestone(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		s := whereItStands(t)
		if !s.Seen() {
			return
		}
		p := review.Preset{
			Rule:      review.RuleRetention,
			Retention: float64(rapid.IntRange(70, 99).Draw(t, "retention")) / 100,
		}
		soon := rapid.IntRange(0, 1000).Draw(t, "sooner")
		late := rapid.IntRange(soon, 1000).Draw(t, "later")
		day := func(days int) time.Time {
			return lastAnswer.Add(time.Duration(days) * 24 * time.Hour)
		}
		if p.Learned(s, day(late)) && !p.Learned(s, day(soon)) {
			t.Fatalf("a card face at a stability of %v counts learned %d days "+
				"out and not %d days out", s.Stability, late, soon)
		}

		lapsed := s
		lapsed.Stability = 0
		if p.Learned(lapsed, day(soon)) {
			t.Fatalf("a card face nothing is known about counts learned at %v", p.Retention)
		}
	})
}
