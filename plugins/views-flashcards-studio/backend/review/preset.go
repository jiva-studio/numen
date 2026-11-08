package review

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
)

// Preset is how the decks pointing at one note are scheduled.
//
// It is read from the frontmatter of a note of `type: preset`. A key the file
type Preset struct {
	Goal Goal
	By   time.Time

	MinutesADay int
	NewADay     int
	ReviewsADay int
	Retention   float64

	Rule     LearnedRule
	Interval int

	Counts BudgetUnit

	Backlog int

	Load       map[time.Weekday]int
	IsEvenLoad bool
}

// FullLoad is a whole day's load, which is what a day the preset does not name
// carries.
const FullLoad = 100

// GetShare is how much of a day's load this day of the week carries, as a share
// of one.
func (p Preset) GetShare(day time.Weekday) float64 {
	per, named := p.Load[day]
	if !named {
		return 1
	}
	return float64(per) / FullLoad
}

// ShouldBalanceLoad reports whether this preset moves a review card off the day the scheduler
// chose.
//
// A preset aiming at a day does not. The pace is what spreads a date's material
// over its days, and the days it has are the days it needs.
func (p Preset) ShouldBalanceLoad() bool { return p.IsEvenLoad && p.Goal != GoalDate }

// GetPlacing is how this preset puts a card on a day, as a short name: whether
// it evens the days out, and the share each day of the week carries. A schedule
// worked out under one placing is not read back under another.
func (p Preset) GetPlacing() string {
	var out strings.Builder
	fmt.Fprintf(&out, "even=%t", p.ShouldBalanceLoad())
	for day := time.Sunday; day <= time.Saturday; day++ {
		fmt.Fprintf(&out, " %s=%d", DayName(day), int(math.Round(p.GetShare(day)*FullLoad)))
	}
	return out.String()
}

// Goal is which value the one control steers.
type Goal string

const (
	// GoalMinutes steers the day by minutes of review.
	GoalMinutes Goal = "minutes_a_day"
	// GoalRetention steers the day by the target retention.
	GoalRetention Goal = "retention"
	// GoalDate steers the day by the date the material is to be learned by.
	GoalDate Goal = "by_date"
)

// IsKnownGoal reports whether a goal is one of the three.
func IsKnownGoal(g Goal) bool {
	switch g {
	case GoalMinutes, GoalRetention, GoalDate:
		return true
	}
	return false
}

// LearnedRule is the criterion by which a preset counts a card as learned. The
// value is stored under the key the rule names, and the other rule keeps its value and takes no part.
type LearnedRule string

const (
	// RuleInterval learns a review card once it is sent away for the preset's
	// interval or longer.
	RuleInterval LearnedRule = "interval"
	// RuleRetention learns a review card once the chance of recalling it today is
	// at or above the preset's retention.
	RuleRetention LearnedRule = "retention"
)

// IsKnownRule reports whether a rule is one of the two.
func IsKnownRule(r LearnedRule) bool {
	switch r {
	case RuleInterval, RuleRetention:
		return true
	}
	return false
}

// IsLearned reports whether a review card standing at this schedule is one the
// person has learned at this instant, under the rule this preset names.
//
// It is the one place the rule is read.
func (p Preset) IsLearned(s Schedule, target time.Time) bool {
	if s.IsNew() {
		return false
	}
	rule, interval, retention := p.GetLearnedRule()
	if rule == RuleRetention {
		return Recall(target.Sub(s.Last), s.Stability) >= retention
	}
	return s.Due.Sub(s.Last) >= time.Duration(interval)*24*time.Hour
}

// GetLearnedRule is the rule a review card is counted learned by here, and the
// two values a rule reads.
//
// A preset naming no rule counts by the default rule, and a value the rule
// to the threshold in Defaults.
func (p Preset) GetLearnedRule() (LearnedRule, int, float64) {
	defaults := Defaults()
	rule, interval, retention := p.Rule, p.Interval, p.Retention
	if !IsKnownRule(rule) {
		rule = defaults.Rule
	}
	if !GetIntervalBounds().Contains(float64(interval)) {
		interval = defaults.Interval
	}
	if !GetRetentionBounds().Contains(retention) {
		retention = defaults.Retention
	}
	return rule, interval, retention
}

// BudgetUnit is what a day's budget is counted in.
//
// Under BudgetUnitCards a review card is charged the first time it is answered in
// a review day and comes round again in it for nothing. Under BudgetUnitShows
// every showing is charged.
type BudgetUnit string

const (
	// BudgetUnitCards counts each review card once a day.
	BudgetUnitCards BudgetUnit = "cards"
	// BudgetUnitShows counts every showing of a review card.
	BudgetUnitShows BudgetUnit = "shows"
)

// IsCharged reports whether a showing of a review card spends a slot of a day's
// count, where isShown reports whether the day has already presented that review card.
//
// It is the one place the counting is read.
func (u BudgetUnit) IsCharged(isShown bool) bool { return u == BudgetUnitShows || !isShown }

// IsKnownBudgetUnit reports whether a value is one of the two.
func IsKnownBudgetUnit(u BudgetUnit) bool {
	switch u {
	case BudgetUnitCards, BudgetUnitShows:
		return true
	}
	return false
}

// Bounds is how far a setting goes, at each end.
type Bounds struct{ Min, Max float64 }

// Contains reports whether a value is within the bounds.
func (b Bounds) Contains(value float64) bool { return value >= b.Min && value <= b.Max }

// What each setting of a preset may be. A day holds no more minutes than it
// has, a retention target outside these is a scheduler asking for what memory
// does not do, and an interval a card is learned at is a day at the least and a
// year at the most.

// GetMinutesADayBounds returns the bounds of the minutes a day.
func GetMinutesADayBounds() Bounds { return Bounds{Min: 0, Max: 24 * 60} }

// GetNewADayBounds returns the bounds of the new cards a day.
func GetNewADayBounds() Bounds { return Bounds{Min: 0, Max: 9999} }

// GetReviewsADayBounds returns the bounds of the reviews a day.
func GetReviewsADayBounds() Bounds { return Bounds{Min: 0, Max: 9999} }

// GetRetentionBounds returns the bounds of the retention target.
func GetRetentionBounds() Bounds { return Bounds{Min: 0.7, Max: 0.99} }

// GetIntervalBounds returns the bounds of the interval a card is learned at, in days.
func GetIntervalBounds() Bounds { return Bounds{Min: 1, Max: 365} }

// GetBacklogBounds returns the bounds of the backlog.
func GetBacklogBounds() Bounds { return Bounds{Min: 0, Max: 100} }

// GetLoadBounds returns the bounds of the load.
func GetLoadBounds() Bounds { return Bounds{Min: 0, Max: FullLoad} }

// Defaults is a preset naming nothing, and how a deck pointing at no preset is
// scheduled.
func Defaults() Preset {
	return Preset{
		Goal:        GoalMinutes,
		MinutesADay: 20,
		NewADay:     10,
		ReviewsADay: 200,
		Retention:   0.9,
		Rule:        RuleInterval,
		Interval:    21,
		Counts:      BudgetUnitCards,
		Backlog:     AllBacklog,
		IsEvenLoad:  true,
		By:          time.Time{},
		Load:        nil,
	}
}

// StopReason is why a preset schedules nothing, and empty where it schedules
// something. The list is closed, and a caller maps a value to a sentence.
type StopReason string

const (
	// StoppedNothing is a preset that schedules: its decks are handed a day of
	// review.
	StoppedNothing StopReason = ""
	// StoppedNoMinutes is a goal of minutes with the minutes at zero.
	StoppedNoMinutes StopReason = "no_minutes"
	// StoppedNoCards is a goal of retention with both card counts at zero.
	StoppedNoCards StopReason = "no_cards"
	// StoppedNoDay is a goal of a date naming no day. The budget the goal names
	// is the day, and a goal that cannot read its own budget schedules nothing.
	StoppedNoDay StopReason = "no_day"
	// StoppedPastDay is a goal of a date whose day is behind us.
	StoppedPastDay StopReason = "past_day"
	// StoppedNoLoad is a day of the week carrying none of the load. It is a
	// fact about one day: the preset schedules on the days that carry some.
	StoppedNoLoad StopReason = "no_load"
	// StoppedNoWeek is a week carrying none of the load. Every day of it has a share
	// of zero, so there is no day on which cards are due.
	StoppedNoWeek StopReason = "no_week"
)

// GetOverallStopReason is why this preset schedules nothing on any day, and
// StoppedNothing where it schedules something.
//
// It is the one place the rule is read. A goal of a date is answered against the
// day holding now, and stops once the day it names is behind that one. A week
// every day of which carries none of the load is read whatever the goal, since
// no budget is spent on a day that schedules nothing.
func (p Preset) GetOverallStopReason(d Day, now time.Time) StopReason {
	switch p.Goal {
	case GoalRetention:
		if p.NewADay == 0 && p.ReviewsADay == 0 {
			return StoppedNoCards
		}
	case GoalDate:
		if p.By.IsZero() {
			return StoppedNoDay
		}
		if p.IsPast(d, now) {
			return StoppedPastDay
		}
	case GoalMinutes:
		if p.MinutesADay == 0 {
			return StoppedNoMinutes
		}
	default:
		if p.MinutesADay == 0 {
			return StoppedNoMinutes
		}
	}
	if p.GetWeeklyShare() == 0 {
		return StoppedNoWeek
	}
	return StoppedNothing
}

// GetWeeklyShare is how many whole days of review a week of this preset holds, counting
// each day of it for the share of the load it carries.
func (p Preset) GetWeeklyShare() float64 {
	out := 0.0
	for day := time.Sunday; day <= time.Saturday; day++ {
		out += p.GetShare(day)
	}
	return out
}

// GetStopReason is why this preset schedules nothing on the day holding now:
// whatever stops the preset at all, and a day of the week carrying none of the
// load. A week with no day carrying any stops the preset itself, so this day is
// one of the quiet days of a week that has loud ones.
func (p Preset) GetStopReason(d Day, now time.Time) StopReason {
	if why := p.GetOverallStopReason(d, now); why != StoppedNothing {
		return why
	}
	if p.GetShare(d.GetDate(now).Weekday()) == 0 {
		return StoppedNoLoad
	}
	return StoppedNothing
}

// IsPaused reports whether the preset schedules nothing.
func (p Preset) IsPaused(d Day, now time.Time) bool {
	return p.GetOverallStopReason(d, now) != StoppedNothing
}

// NeverClears is a pace that leaves something overdue on every day projected.
const NeverClears = -1

// NeverLearns is a day further off than what was looked at: a projection whose
// horizon ends with one review card still to learn.
const NeverLearns = -1

// LearnsUnasked is a projection with no day on which the whole material is
// learned.
const LearnsUnasked = -2

func (p Preset) getPace(d Day, now time.Time, unbegunCards, daysToLearn int) int {
	reviewDays := p.getReviewDays(d, now)
	if reviewDays <= 0 {
		return 0
	}
	if daysToLearn == NeverLearns {
		return unbegunCards
	}
	in := p.getCapacityToBegin(d, now, daysToLearn)
	if in <= 0 {
		return unbegunCards
	}
	return int(math.Ceil(float64(unbegunCards) / in))
}

func (p Preset) getCapacityToBegin(d Day, now time.Time, daysToLearn int) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(DayLayout, d.GetName(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	span := int(to.Sub(from).Hours()/24) + 1
	for span > 0 && daysToLearn > 0 {
		span--
		if p.GetShare(from.AddDate(0, 0, span).Weekday()) > 0 {
			daysToLearn--
		}
	}
	return p.getDaysOfReview(from, span)
}

func (p Preset) getReviewDays(d Day, now time.Time) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(DayLayout, d.GetName(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	return p.getDaysOfReview(from, int(to.Sub(from).Hours()/24)+1)
}

func (p Preset) getDaysOfReview(from time.Time, days int) float64 {
	if days <= 0 {
		return 0
	}
	whole := days / 7
	out := p.GetWeeklyShare() * float64(whole)
	for i := range days % 7 {
		out += p.GetShare(from.AddDate(0, 0, whole*7+i).Weekday())
	}
	return out
}

// IsPast reports whether the review day the goal names is behind us. The day it
// names is a whole day of review, and a preset aiming at no day is never past.
func (p Preset) IsPast(d Day, now time.Time) bool {
	if p.Goal != GoalDate || p.By.IsZero() {
		return false
	}
	return !now.Before(d.GetEndOfDate(p.By))
}

func readGoal(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["goal"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "goal is not text")
	case IsKnownGoal(Goal(name)):
		p.Goal = Goal(name)
	default:
		*problems = append(*problems, "goal "+name+" is not minutes_a_day, retention or by_date")
	}
}

func readLearnedRule(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["learned"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "learned is not text")
	case IsKnownRule(LearnedRule(name)):
		p.Rule = LearnedRule(name)
	default:
		*problems = append(*problems, "learned "+name+" is not interval or retention")
	}
}

func readBudgetUnit(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["counts"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "counts is not text")
	case IsKnownBudgetUnit(BudgetUnit(name)):
		p.Counts = BudgetUnit(name)
	default:
		*problems = append(*problems, "counts "+name+" is not cards or shows")
	}
}

func readByDate(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["by_date"]
	if !present || raw == nil {
		if p.Goal == GoalDate && p.By.IsZero() {
			*problems = append(*problems, "a preset targeting a day requires the day, under by_date")
		}
		return
	}
	switch value := raw.(type) {
	case time.Time:
		p.By = value
	case string:
		day, err := time.Parse(DayLayout, strings.TrimSpace(value))
		if err != nil {
			*problems = append(*problems, "by_date "+value+" is not a day, written as "+DayLayout)
			break
		}
		p.By = day
	default:
		*problems = append(*problems, "by_date is not a day")
	}
	if p.Goal == GoalDate && p.By.IsZero() {
		*problems = append(*problems, "a preset targeting a day requires the day, under by_date")
	}
}

func readSchedule(front map[string]any, p *Preset, problems *[]string) {
	readGoal(front, p, problems)
	readLearnedRule(front, p, problems)
	readBudgetUnit(front, p, problems)
	readByDate(front, p, problems)
}

func readLimits(front map[string]any, p *Preset, problems *[]string) {
	p.MinutesADay = readWholeNumber(front, "minutes_a_day", GetMinutesADayBounds(), p.MinutesADay, problems)
	p.NewADay = readWholeNumber(front, "new_a_day", GetNewADayBounds(), p.NewADay, problems)
	p.ReviewsADay = readWholeNumber(front, "reviews_a_day", GetReviewsADayBounds(), p.ReviewsADay, problems)
	p.Backlog = readWholeNumber(front, "backlog", GetBacklogBounds(), p.Backlog, problems)
	p.Interval = readWholeNumber(front, "interval", GetIntervalBounds(), p.Interval, problems)
}

func readRetention(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["retention"]
	if !present || raw == nil {
		return
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, "retention is not a number")
	case !GetRetentionBounds().Contains(value):
		*problems = append(*problems, fmt.Sprintf(
			"retention %g is outside %g to %g", value, GetRetentionBounds().Min, GetRetentionBounds().Max))
	default:
		p.Retention = value
	}
}

func readEvenLoad(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["even_load"]
	if !present || raw == nil {
		return
	}
	value, isBool := raw.(bool)
	if !isBool {
		*problems = append(*problems, "even_load is neither true nor false")
		return
	}
	p.IsEvenLoad = value
}

func readDayShare(name string, raw any, p *Preset, daysLen int, problems *[]string) {
	day, recognized := Weekday(name)
	if !recognized {
		*problems = append(*problems, "the load of "+name+" is not a day of the week")
		return
	}
	share, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, "the load of "+name+" is not a number")
	case share != math.Trunc(share):
		*problems = append(*problems, "the load of "+name+" is counted in whole per cent")
	case !GetLoadBounds().Contains(share):
		*problems = append(*problems, fmt.Sprintf("the load of %s, %g, is outside %g to %g",
			name, share, GetLoadBounds().Min, GetLoadBounds().Max))
	default:
		if p.Load == nil {
			p.Load = make(map[time.Weekday]int, daysLen)
		}
		p.Load[day] = int(share)
	}
}

func readLoad(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["load"]
	if !present || raw == nil {
		return
	}
	days, isMapping := raw.(map[string]any)
	if !isMapping {
		*problems = append(*problems, "load is a day of the week against a share of a day's load")
		return
	}
	// A map iterates in arbitrary order, so the keys are sorted and a note read
	// twice yields the same result.
	for _, name := range slices.Sorted(maps.Keys(days)) {
		readDayShare(name, days[name], p, len(days), problems)
	}
}

func readWeights(front map[string]any, p *Preset, problems *[]string) {
	readRetention(front, p, problems)
	readEvenLoad(front, p, problems)
	readLoad(front, p, problems)
}

// ReadPreset reads a preset from frontmatter and reports the problems found.
func ReadPreset(front map[string]any) (Preset, []string) {
	p := Defaults()
	var problems []string

	readSchedule(front, &p, &problems)
	readLimits(front, &p, &problems)
	readWeights(front, &p, &problems)

	return p, problems
}

// Weekday is the day of the week a preset writes as the name, and whether the name is one.
func Weekday(name string) (time.Weekday, bool) {
	for day := time.Sunday; day <= time.Saturday; day++ {
		if DayName(day) == strings.ToLower(strings.TrimSpace(name)) {
			return day, true
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="Constant time.Sunday is already the zero Weekday value."
	return time.Sunday, false
}

// DayName is how a preset writes a day of the week.
func DayName(day time.Weekday) string {
	return strings.ToLower(day.String()[:3])
}

func readWholeNumber(
	front map[string]any, key string, bounds Bounds, fallback int, problems *[]string,
) int {
	raw, present := front[key]
	if !present || raw == nil {
		return fallback
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, key+" is not a number")
	case value != math.Trunc(value):
		*problems = append(*problems, key+" is counted in whole numbers")
	case !bounds.Contains(value):
		*problems = append(*problems, fmt.Sprintf(
			"%s %g is outside %g to %g", key, value, bounds.Min, bounds.Max))
	default:
		return int(value)
	}
	return fallback
}

func number(raw any) (float64, bool) {
	switch value := raw.(type) {
	case int:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}
