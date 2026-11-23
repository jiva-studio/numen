package projection

import (
	"context"
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

// One card a day overdue is answered on the day the run opens, sent away for two days, answered
// again on the third day and sent away past the end of the run.
func TestARunAnswersTheCardsAsTheyFallDue(t *testing.T) {
	t.Parallel()
	got, err := newTestSimulation(4).Run(t.Context(), getTestNow(), getRetentionSettings(10, 10), getOverdueCards(1), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Load, []int{1, 0, 1, 0}) {
		t.Errorf("the days answer %v, want [1 0 1 0]", got.Load)
	}
	if got.Answered != 2 || got.Seen != 1 || got.ReviewCards != 1 {
		t.Errorf("answered %d, seen %d of %d cards, want 2, 1 of 1", got.Answered, got.Seen, got.ReviewCards)
	}
	if got.Clears != 1 {
		t.Errorf("the overdue card clears in %d days, want 1", got.Clears)
	}
	if got.Learned != 0 || got.Learns != 3 {
		t.Errorf("learned %d, learns in %d days, want 0 and 3", got.Learned, got.Learns)
	}
	if got.Through[0] != 0 || got.Through[3] != 1 {
		t.Errorf("the share learned runs %v, want 0 on the first day and 1 on the last", got.Through)
	}
}

func TestEveryAnswerIsPlacedByTheLoad(t *testing.T) {
	t.Parallel()
	placed := 0
	run := newTestSimulation(4)
	run.By = fakeScheduler{placed: &placed}
	got, err := run.Run(t.Context(), getTestNow(), getRetentionSettings(10, 10), getOverdueCards(3), 2)
	if err != nil {
		t.Fatal(err)
	}
	if placed != got.Answered {
		t.Errorf("%d answers were placed, and the run answered %d", placed, got.Answered)
	}
}

func TestAPausedPresetAnswersNothing(t *testing.T) {
	t.Parallel()
	p := Settings{Goal: GoalMinutes, MinutesADay: 0, NewADay: 10, ReviewsADay: 10, Rule: RuleInterval, Interval: 4}
	got, err := newTestSimulation(2).Run(t.Context(), getTestNow(), p, getOverdueCards(3), 4)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Load, []int{0, 0}) || !slices.Equal(got.Admitted, []bool{false, false}) {
		t.Errorf("a paused preset answers %v on days admitted %v", got.Load, got.Admitted)
	}
	if !slices.Equal(got.Backlog, []int{3, 3}) {
		t.Errorf("the backlog runs %v, want the three overdue cards every day", got.Backlog)
	}
	if !got.Closed[0].Has(ClosedPaused) {
		t.Errorf("a paused day is closed by %v", got.Closed[0])
	}
	if _, isAdmitted := got.FindSession(); isAdmitted {
		t.Error("a paused run has a session")
	}
}

func TestADayStopsAtItsMinutes(t *testing.T) {
	t.Parallel()
	p := Settings{Goal: GoalMinutes, MinutesADay: 3, Rule: RuleInterval, Interval: 4, Backlog: wholeBacklog}
	got, err := newTestSimulation(1).Run(t.Context(), getTestNow(), p, getOverdueCards(5), 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Load[0] != 3 || got.Spent[0] != 3*time.Minute || got.Backlog[0] != 2 {
		t.Errorf("a day of three minutes answers %d in %v and leaves %d, want 3 in 3m and 2",
			got.Load[0], got.Spent[0], got.Backlog[0])
	}
	if !slices.Equal(got.Closed[0], BudgetNames{ClosedMinutes}) {
		t.Errorf("the day is closed by %v, want the minutes", got.Closed[0])
	}
}

func TestADayStopsAtBothItsCounts(t *testing.T) {
	t.Parallel()
	got, err := newTestSimulation(1).Run(t.Context(), getTestNow(), getRetentionSettings(1, 2), getOverdueCards(5), 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.Load[0] != 3 || got.Reviewed[0] != 3 || got.Seen != 6 {
		t.Errorf("the day answers %d, reviews %d and has seen %d, want 3, 3 and 6", got.Load[0], got.Reviewed[0], got.Seen)
	}
	if !slices.Equal(got.Closed[0], BudgetNames{ClosedNew, ClosedReviews}) {
		t.Errorf("the day is closed by %v, want both counts", got.Closed[0])
	}
}

// A card forgotten on every answer is returned within the day. Counted in cards it is one card
// however often it is shown, and counted in showings each showing spends a slot.
func TestACountOfShowingsChargesEveryShowing(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		isEveryShowingCharged bool
		load                  int
		closed                BudgetNames
	}{
		{isEveryShowingCharged: false, load: MostShowings, closed: BudgetNames{}},
		{isEveryShowingCharged: true, load: 1, closed: BudgetNames{ClosedReviews}},
	} {
		run := newTestSimulation(1)
		run.Recalls = func(Card, time.Time) float64 { return 0 }
		p := getRetentionSettings(0, 1)
		p.IsEveryShowingCharged = one.isEveryShowingCharged
		got, err := run.Run(t.Context(), getTestNow(), p, getOverdueCards(1), 0)
		if err != nil {
			t.Fatal(err)
		}
		if got.Load[0] != one.load || !slices.Equal(got.Closed[0], one.closed) {
			t.Errorf("charging every showing %v: the day shows %d and closes on %v, want %d and %v",
				one.isEveryShowingCharged, got.Load[0], got.Closed[0], one.load, one.closed)
		}
	}
}

func TestTheShareRecalledIsReportedOnTheDaysAsked(t *testing.T) {
	t.Parallel()
	run := newTestSimulation(3)
	run.Retains = []int{-1, 0, 2, 3}
	got, err := run.Run(t.Context(), getTestNow(), getRetentionSettings(10, 10), getOverdueCards(2), 0)
	if err != nil {
		t.Fatal(err)
	}
	for day, want := range []bool{true, false, true, false} {
		if _, isReported := got.Retained.GetShare(day); isReported != want {
			t.Errorf("day %d reports a share: %v, want %v", day, isReported, want)
		}
	}
	if share, isReported := got.Retained.GetShare(0); !isReported || share <= 0 || share > 1 {
		t.Errorf("the share on the first day is %v, reported %v", share, isReported)
	}
	if _, isReported := got.Retained.GetShare(1); isReported {
		t.Error("a day not asked for reports a share")
	}
}

func TestMaterialAlreadyLearnedIsLearnedInNoDays(t *testing.T) {
	t.Parallel()
	last := getTestOpen().Add(-oneDay)
	learned := []Card{{Due: last.Add(10 * oneDay), Last: last, Reps: 3, Stability: 10, Phase: 2}}
	got, err := newTestSimulation(2).Run(t.Context(), getTestNow(), getRetentionSettings(10, 10), learned, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.Learned != 1 || got.Learns != 0 || got.Clears != 0 {
		t.Errorf("learned %d, learns in %d, clears in %d, want 1, 0 and 0", got.Learned, got.Learns, got.Clears)
	}
}

func TestARunStopsWhenItsCallerCancels(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := newTestSimulation(3).Run(ctx, getTestNow(), getRetentionSettings(10, 10), getOverdueCards(1), 0)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled run returns %v", err)
	}
}

// A card answered under a chance of recall of one half moves half way between the two endings.
func TestTheChanceOfRecallWeighsTheEndings(t *testing.T) {
	t.Parallel()
	run := newTestSimulation(1)
	run.Recalls = func(Card, time.Time) float64 { return 0.5 }
	open := getTestOpen()
	c := Card{Due: open, Last: open.Add(-2 * oneDay), Reps: 1, Stability: 2, Phase: 2}
	got := run.step(c, open, getRetentionSettings(1, 1), nil)
	if got.Stability != 2.5 || got.Difficulty != 0.5 || !got.Due.Equal(open.Add(2*oneDay+30*time.Minute)) {
		t.Errorf("the card is %+v, want stability 2.5, difficulty 0.5, due in 2 days 30 minutes", got)
	}
}

func TestAStabilityOfZeroIsRaisedToTheLeast(t *testing.T) {
	t.Parallel()
	run := newTestSimulation(1)
	run.Recalls = nil
	open := getTestOpen()
	got := run.step(Card{Due: open, Last: open.Add(-oneDay), Reps: 1, Stability: 0, Phase: 2}, open, getRetentionSettings(1, 1), nil)
	if math.IsNaN(got.Stability) || got.Stability <= 0 || !got.Due.After(open) {
		t.Errorf("a card at no stability steps to %+v", got)
	}
}

func TestAnEvenLoadMovesACardOffACrowdedDay(t *testing.T) {
	t.Parallel()
	run := newTestSimulation(1)
	open := getTestOpen()
	c := Card{Due: open, Last: open.Add(-2 * oneDay), Reps: 1, Stability: 2, Phase: 2}
	for _, one := range []struct {
		isEvenLoad bool
		want       time.Time
	}{
		{isEvenLoad: false, want: open.Add(4 * oneDay)},
		{isEvenLoad: true, want: open.Add(3 * oneDay)},
	} {
		on := run.By.NewLoad()
		on.Add(open.Add(4 * oneDay))
		p := getRetentionSettings(1, 1)
		p.IsEvenLoad = one.isEvenLoad
		if got := run.step(c, open, p, on); !got.Due.Equal(one.want) {
			t.Errorf("even load %v: the card is due %v, want %v", one.isEvenLoad, got.Due, one.want)
		}
	}
}

func TestDaysToLearnACardBegunNow(t *testing.T) {
	t.Parallel()
	by, d := fakeScheduler{placed: nil}, utcDays{}
	p := getRetentionSettings(1, 1)
	p.Interval = 1
	if got := GetDaysToLearn(by, by, d, p, getTestNow()); got != 0 {
		t.Errorf("a card sent away for a day on its first answer is learned at an interval of 1 in %d days", got)
	}
	p.Interval = 4
	four := GetDaysToLearn(by, by, d, p, getTestNow())
	if four <= 0 || four == NeverLearns {
		t.Errorf("a card is learned at an interval of 4 in %d days", four)
	}
	p.Interval = 8
	if eight := GetDaysToLearn(by, by, d, p, getTestNow()); eight < four {
		t.Errorf("a card is learned at an interval of 8 in %d days and at 4 in %d", eight, four)
	}
	p.Load = map[time.Weekday]int{
		time.Sunday: 0, time.Monday: 0, time.Tuesday: 0, time.Wednesday: 0, time.Thursday: 0, time.Friday: 0, time.Saturday: 0,
	}
	if got := GetDaysToLearn(by, by, d, p, getTestNow()); got != NeverLearns {
		t.Errorf("a week carrying no load learns a card in %d days", got)
	}
}

func TestWhatADateCannotReachIsShort(t *testing.T) {
	t.Parallel()
	run, open := newTestSimulation(1), getTestOpen()
	p := Settings{Goal: GoalDate, By: open.AddDate(0, 0, 2), Rule: RuleInterval, Interval: 30}
	if got := run.countShort(p, getOverdueCards(2), 3, open); got != 5 {
		t.Errorf("%d of five cards are short of an interval of 30 in three days", got)
	}
	p.Interval = 1
	if got := run.countShort(p, getOverdueCards(2), 3, open); got != 0 {
		t.Errorf("%d cards are short of an interval of 1 in three days", got)
	}
	p.By = time.Time{}
	if got := run.countShort(p, getOverdueCards(2), 3, open); got != 0 {
		t.Errorf("a date naming no day leaves %d short", got)
	}
	p.Goal, p.By = GoalMinutes, open.AddDate(0, 0, 2)
	if got := run.countShort(p, getOverdueCards(2), 3, open); got != 0 {
		t.Errorf("a goal of minutes leaves %d short", got)
	}
}

func TestCountOverdue(t *testing.T) {
	t.Parallel()
	open := getTestOpen()
	later := Card{Due: open.Add(20 * time.Hour), Last: open.Add(-oneDay), Reps: 1, Stability: 1}
	cards := append(getOverdueCards(2), later, Card{})
	if got := CountOverdue(utcDays{}, cards, getTestNow()); got != 2 {
		t.Errorf("%d cards are overdue, want the 2 whose day has passed", got)
	}
}

func TestBudgetNamesAreNamedOnceInOrder(t *testing.T) {
	t.Parallel()
	got := BudgetNames{}.Add(ClosedReviews).Add(ClosedNew).Add(ClosedReviews).Add(ClosedNothing)
	if !slices.Equal(got, BudgetNames{ClosedNew, ClosedReviews}) {
		t.Errorf("the budgets read %v, want [new reviews]", got)
	}
}
