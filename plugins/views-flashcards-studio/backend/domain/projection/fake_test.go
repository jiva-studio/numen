package projection

import (
	"math"
	"time"
)

const (
	oneDay       = 24 * time.Hour
	wholeBacklog = 100
	fullLoad     = 100
)

// utcDays is a calendar whose review days begin at midnight UTC.
type utcDays struct{}

func (utcDays) GetStart(at time.Time) time.Time {
	y, m, d := at.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (c utcDays) GetEnd(at time.Time) time.Time { return c.GetStart(at).Add(oneDay) }

func (utcDays) GetName(at time.Time) string { return at.UTC().Format(time.DateOnly) }

func (utcDays) GetEndOfDate(date time.Time) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}

// fakeScheduler doubles the days a card is sent away for on every recalled answer, brings a
// forgotten card back an hour later, and admits a day by the settings alone.
type fakeScheduler struct {
	placed *int
}

func (fakeScheduler) GetEndings(c Card, at time.Time) (recalled, forgotten Card) {
	away := math.Max(2*c.Stability, 1)
	recalled = Card{
		Due: at.Add(time.Duration(away * float64(oneDay))), Last: at, Reps: c.Reps + 1, Lapses: c.Lapses,
		Stability: away, Difficulty: c.Difficulty, Phase: 2,
	}
	forgotten = Card{
		Due: at.Add(time.Hour), Last: at, Reps: c.Reps + 1, Lapses: c.Lapses + 1,
		Stability: 1, Difficulty: c.Difficulty + 1, Phase: 3,
	}
	return recalled, forgotten
}

func (fakeScheduler) GetRecall(c Card, at time.Time) float64 {
	if c.Stability <= 0 {
		return 0
	}
	return math.Pow(0.9, at.Sub(c.Last).Hours()/24/c.Stability)
}

func (s fakeScheduler) IsLearned(p Settings, c Card, at time.Time) bool {
	if c.IsNew() {
		return false
	}
	if p.Rule == RuleRetention {
		return s.GetRecall(c, at) >= p.Retention
	}
	return c.Due.Sub(c.Last) >= time.Duration(p.Interval)*oneDay
}

func (s fakeScheduler) GetAllowance(p Settings, open time.Time, spent Spent, unbegunCards, _ int) Allowance {
	share := s.GetShare(p, open.Weekday())
	out := Allowance{
		New:      int(math.Round(share*float64(p.NewADay))) - spent.New,
		Reviews:  int(math.Round(share*float64(p.ReviewsADay))) - spent.Reviews,
		Minutes:  time.Duration(share*float64(p.MinutesADay)*float64(time.Minute)) - spent.Took,
		Limits:   Limits{New: ClosedNothing, Reviews: ClosedNothing, Minutes: ClosedMinutes},
		Backlog:  p.Backlog,
		IsPaused: share == 0,
	}
	switch p.Goal {
	case GoalRetention:
		out.Limits = Limits{New: ClosedNew, Reviews: ClosedReviews, Minutes: ClosedNothing}
		out.Backlog = wholeBacklog
	case GoalDate:
		left := max(1, int(p.By.Sub(open)/oneDay)+1)
		out.New = (unbegunCards + left - 1) / left
		out.Limits = Limits{New: ClosedDate, Reviews: ClosedNothing, Minutes: ClosedNothing}
		out.Backlog = wholeBacklog
		out.IsPaused = out.IsPaused || p.By.IsZero()
	case GoalMinutes:
		out.IsPaused = out.IsPaused || p.MinutesADay == 0
	}
	return out
}

func (s fakeScheduler) NewLoad() Load { return &fakeLoad{on: make(map[int]int), placed: s.placed} }

func (fakeScheduler) GetShare(p Settings, day time.Weekday) float64 {
	per, named := p.Load[day]
	if !named {
		return 1
	}
	return float64(per) / fullLoad
}

func (fakeScheduler) IsNextCardDebt(a Allowance, debt, begun int, isDue, isNew bool) bool {
	if !isDue {
		return false
	}
	if !isNew {
		return true
	}
	return debt*wholeBacklog < a.Backlog*(debt+begun+1)
}

// fakeLoad moves a card to the day before its due day when that day holds fewer cards and the
// settings even the load out.
type fakeLoad struct {
	on     map[int]int
	placed *int
}

func (l *fakeLoad) Add(due time.Time) { l.on[l.GetDayNumber(due)]++ }

func (l *fakeLoad) GetDayNumber(at time.Time) int { return int(at.Unix() / int64(oneDay/time.Second)) }

func (l *fakeLoad) PlaceCard(p Settings, at, due time.Time) time.Time {
	if l.placed != nil {
		*l.placed++
	}
	number := l.GetDayNumber(due)
	if p.IsEvenLoad && number-1 > l.GetDayNumber(at) && l.on[number-1] < l.on[number] {
		due = due.Add(-oneDay)
	}
	l.Add(due)
	return due
}

func getTestNow() time.Time { return time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC) }

func getTestOpen() time.Time { return utcDays{}.GetStart(getTestNow()) }

func newTestSimulation(days int) Simulation {
	return Simulation{
		By: fakeScheduler{placed: nil}, Rules: fakeScheduler{placed: nil}, Day: utcDays{}, Cost: Cost{New: time.Minute, Review: time.Minute},
		Days: days, Retains: nil, Recalls: GetFullRecall, Spent: Spent{},
	}
}

func getOverdueCards(count int) []Card {
	out := make([]Card, count)
	for i := range out {
		last := getTestOpen().Add(-2 * oneDay)
		out[i] = Card{Due: last.Add(oneDay), Last: last, Reps: 1, Stability: 1, Phase: 2}
	}
	return out
}

func getRetentionSettings(newADay, reviewsADay int) Settings {
	return Settings{
		Goal: GoalRetention, NewADay: newADay, ReviewsADay: reviewsADay, Retention: 0.9,
		Rule: RuleInterval, Interval: 4, Backlog: wholeBacklog,
	}
}
