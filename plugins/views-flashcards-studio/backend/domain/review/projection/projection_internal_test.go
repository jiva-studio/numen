package projection

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

func TestProjectionAndCurveInternals(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := Day{Starts: 0, In: time.UTC}
	errBoom := errors.New("boom")

	if Points != 25 || MostAhead != 5*365 || LeastCeiling != 60 {
		t.Fatalf("unexpected curve/grid constants: Points=%d MostAhead=%d LeastCeiling=%d",
			Points, MostAhead, LeastCeiling)
	}
	if got := GetOutsideGrid(); got != (CurvePoint{Index: -1, Value: 0, Day: ""}) {
		t.Fatalf("unexpected GetOutsideGrid: %+v", got)
	}
	if got := getMinutesUpperBound(1, 1); got != 60 {
		t.Fatalf("expected LeastCeiling 60 from getMinutesUpperBound(1,1), got %g", got)
	}

	sim := Simulation{By: NewFSRSAt(0.9), Day: day, Cost: review.GetDefaultCost(), Days: 5}
	for _, goal := range []Goal{GoalMinutes, GoalRetention, GoalDate} {
		p := review.Defaults()
		p.Goal = goal
		p.By = time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
		if _, err := sim.Curve(ctx, now, p, CurveInputs{
			Places: func(int, func(int) error) error { return errBoom },
		}); !errors.Is(err, errBoom) {
			t.Fatalf("expected Places error for goal %s, got %v", goal, err)
		}
		got, err := sim.Curve(ctx, now, p, CurveInputs{UnbegunCount: 2})
		if err != nil || len(got.Grid) == 0 {
			t.Fatalf("expected non-empty Grid for goal %s, got len=%d err=%v", goal, len(got.Grid), err)
		}
	}

	// Test mergeByDue when two schedules have equal Due, Last, Stability, Difficulty (<= 0 prefers carried).
	sEqual := Schedule{Due: now, Last: now.Add(-time.Hour), Stability: 2, Difficulty: 5}
	merged := mergeByDue(nil, []int{0}, []int{1}, []Schedule{sEqual, sEqual})
	if !slices.Equal(merged, []int{0, 1}) {
		t.Fatalf("expected carried index 0 before fell index 1 on tie, got %v", merged)
	}

	// Test compareByDue when only Difficulty differs.
	sDiffLow := Schedule{Due: now, Last: now, Stability: 2, Difficulty: 3}
	sDiffHigh := Schedule{Due: now, Last: now, Stability: 2, Difficulty: 7}
	if compareByDue(sDiffLow, sDiffHigh) >= 0 || compareByDue(sDiffHigh, sDiffLow) <= 0 {
		t.Fatal("expected compareByDue to compare Difficulty when Due, Last, and Stability match")
	}

	// Test learnedCount: countBegun when !isCarried and when isCarried && isLearned;
	// countLearned when isCarried && !wanted, when !isCarried with new card vs learned card, and when reviewCards == 0.
	pRet := review.Defaults()
	pRet.Rule = review.RuleRetention
	pRet.Retention = 0.9
	lcNotCarried := newLearnedCount(pRet, nil, now)
	lcNotCarried.countBegun(Schedule{Last: now, Due: now.Add(30 * 24 * time.Hour), Stability: 100}, now)
	if lcNotCarried.learned != nil || lcNotCarried.count != 0 {
		t.Fatalf("expected countBegun no-op when !isCarried, got %+v", lcNotCarried)
	}

	pInt := review.Defaults()
	pInt.Rule = RuleInterval
	pInt.Interval = 1
	lcCarried := newLearnedCount(pInt, nil, now)
	lcCarried.countBegun(Schedule{Last: now, Due: now.Add(48 * time.Hour), Stability: 10}, now)
	if lcCarried.count != 1 || !slices.Equal(lcCarried.learned, []bool{true}) {
		t.Fatalf("expected countBegun to increment count when learned under RuleInterval, got %+v", lcCarried)
	}
	if stands, back := lcCarried.countLearned(
		[]Schedule{{Last: now, Stability: 10}}, now, 1, false,
	); stands != 1 || back != 0 {
		t.Fatalf("expected back=0 when !wanted on carried learnedCount, got stands=%d back=%g", stands, back)
	}

	lcZeroTarget := &learnedCount{preset: pRet, isCarried: false, target: 0}
	if stands, _ := lcZeroTarget.countLearned([]Schedule{{}}, now, 1, true); stands != 0 {
		t.Fatalf("expected unanswered card (IsNew) to not count in stands even when target==0, got %d", stands)
	}
	lcExact := &learnedCount{preset: pRet, isCarried: false, target: review.Recall(24*time.Hour, 1.0)}
	sExact := Schedule{Last: now.Add(-24 * time.Hour), Due: now, Stability: 1.0}
	if stands, back := lcExact.countLearned([]Schedule{sExact}, now, 0, true); stands != 1 || back != 0 {
		t.Fatalf("expected stands=1 and back=0 when reviewCards==0 and one == target, got stands=%d back=%g",
			stands, back)
	}
}

func TestSimulationInternals(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	day := Day{Starts: 0, In: time.UTC}

	if Ahead != 90 || LeastStability != 1.0/(24*60) || MostShowings != 8 || LongestLearning != 10*365 || mostAnswers != 1000 {
		t.Fatalf("unexpected simulation constants: Ahead=%d LeastStability=%g MostShowings=%d LongestLearning=%d mostAnswers=%d",
			Ahead, LeastStability, MostShowings, LongestLearning, mostAnswers)
	}
	if newNoCost() != (AnswerCost{}) {
		t.Fatalf("expected zero AnswerCost, got %+v", newNoCost())
	}
	if newNoSpent() != (Spent{}) {
		t.Fatalf("expected zero Spent, got %+v", newNoSpent())
	}
	if newNoSchedule() != (Schedule{}) {
		t.Fatalf("expected zero Schedule, got %+v", newNoSchedule())
	}
	if createEmptyAllowance() != (Allowance{
		Limits: review.Limits{
			New:     ClosedNothing,
			Reviews: ClosedNothing,
			Minutes: ClosedNothing,
			Backlog: ClosedNothing,
		},
	}) {
		t.Fatalf("unexpected createEmptyAllowance: %+v", createEmptyAllowance())
	}

	sim := Simulation{By: NewFSRSAt(0.9), Day: day, Cost: AnswerCost{New: time.Minute, Review: time.Minute}}
	pNoLoad := review.Defaults()
	pNoLoad.Load = map[time.Weekday]int{
		time.Sunday: 0, time.Monday: 0, time.Tuesday: 0, time.Wednesday: 0,
		time.Thursday: 0, time.Friday: 0, time.Saturday: 0,
	}
	if got := GetDaysToLearn(sim.By, day, pNoLoad, now); got != NeverLearns {
		t.Fatalf("expected NeverLearns when weekly load is 0, got %d", got)
	}

	// Test getScheduleAfterShowing when !c.IsNew() && !c.Due.Before(ends) vs c.IsNew() with Due >= ends.
	open := day.GetStart(now)
	ends := day.GetEnd(open)
	futureSched := Schedule{Last: open.Add(-24 * time.Hour), Due: ends.Add(24 * time.Hour), Reps: 1, Stability: 5, Difficulty: 5}
	if got := sim.getScheduleAfterShowing(futureSched, open, ends, review.Defaults(), nil); got != futureSched {
		t.Fatalf("expected answered card due after ends to be unchanged, got %+v", got)
	}
	newWithFutureDue := Schedule{Last: time.Time{}, Due: ends.Add(24 * time.Hour)}
	if got := sim.getScheduleAfterShowing(newWithFutureDue, open, ends, review.Defaults(), nil); got.IsNew() {
		t.Fatalf("expected new card to be stepped even if Due >= ends, got %+v", got)
	}

	// Test getScheduleAfterDay stopping once !c.IsNew() && !s.Day.IsDue(c, open), and stepping when c.IsNew() with future Due.
	if got := sim.getScheduleAfterDay(newWithFutureDue, open, review.Defaults()); got.IsNew() {
		t.Fatalf("expected new card with future Due to be stepped in getScheduleAfterDay, got %+v", got)
	}
	steppedOnce := sim.getScheduleAfterDay(Schedule{}, open, review.Defaults())
	if steppedOnce.Reps >= MostShowings {
		t.Fatalf("expected getScheduleAfterDay to break once card is scheduled beyond today, got Reps=%d", steppedOnce.Reps)
	}

	// Test canReachGoal: card due after delta breaks immediately; card due on a later day jumps open to c.Due;
	// new card with future Due still steps; zero load skips getScheduleAfterDay.
	delta := open.AddDate(0, 0, 10)
	pEasyInt := review.Defaults()
	pEasyInt.Rule = RuleInterval
	pEasyInt.Interval = 1
	pEasyInt.Goal = GoalDate
	pEasyInt.By = delta

	unlearnedFuture := Schedule{Last: delta.Add(12 * time.Hour), Due: delta.Add(24 * time.Hour), Stability: 0.5, Difficulty: 5}
	if sim.canReachGoal(pEasyInt, unlearnedFuture, open, delta) {
		t.Fatal("expected card due after delta with <1d interval to break without further review and return false")
	}
	if !sim.canReachGoal(pEasyInt, newWithFutureDue, open, delta) {
		t.Fatal("expected new card with future Due to still be reviewed and reach 1-day interval goal")
	}
	pNoLoadGoal := pEasyInt
	pNoLoadGoal.Load = pNoLoad.Load
	if sim.canReachGoal(pNoLoadGoal, Schedule{}, open, delta) {
		t.Fatal("expected canReachGoal false when all weekday loads are 0")
	}
	// Verify jumping open = s.Day.GetStart(c.Due) allows a card requiring multiple multi-day reviews to reach a 30-day interval over 365 days
	// in far fewer than mostAnswers steps (without jumping open to c.Due, 1000 steps would still be needed if days were walked 1 by 1 beyond 1000 days).
	pLong := review.Defaults()
	pLong.Rule = RuleInterval
	pLong.Interval = 100
	pLong.Goal = GoalDate
	farDelta := open.AddDate(0, 0, 2000)
	pLong.By = farDelta
	if !sim.canReachGoal(pLong, Schedule{}, open, farDelta) {
		t.Fatal("expected card to reach 100-day interval over 2000 days by jumping open to c.Due")
	}

	// Test short: p.Goal != GoalDate, p.By.IsZero(), unreviewed == 0 vs unreviewed > 0, and += vs = when both cards and unreviewed fail.
	if got := sim.short(review.Defaults(), []Schedule{{}}, 5, open); got != 0 {
		t.Fatalf("expected 0 from short when Goal != GoalDate, got %d", got)
	}
	pZeroBy := pEasyInt
	pZeroBy.By = time.Time{}
	if got := sim.short(pZeroBy, []Schedule{{}}, 5, open); got != 0 {
		t.Fatalf("expected 0 from short when By is zero, got %d", got)
	}
	pImpossible := review.Defaults()
	pImpossible.Goal = GoalDate
	pImpossible.By = open
	pImpossible.Rule = RuleInterval
	pImpossible.Interval = 365
	if got := sim.short(pImpossible, []Schedule{unlearnedFuture}, 0, open); got != 1 {
		t.Fatalf("expected 1 from short when 1 card fails and unreviewed==0, got %d", got)
	}
	if got := sim.short(pImpossible, []Schedule{unlearnedFuture}, 3, open); got != 4 {
		t.Fatalf("expected 1+3=4 from short when 1 card and 3 unreviewed fail, got %d", got)
	}

	// Test step: LeastStability floor on existing card, AssignDueDay on new card, and away*float64(time.Second) on existing card.
	tinyStab := Schedule{Last: open.Add(-24 * time.Hour), Due: open, Reps: 1, Stability: 1e-9, Difficulty: 5, Phase: 2}
	simZeroRecall := sim
	simZeroRecall.Recalls = func(Schedule, time.Time) float64 { return 0.5 }
	steppedTiny := simZeroRecall.step(tinyStab, open, review.Defaults(), nil)
	if math.IsNaN(steppedTiny.Stability) || steppedTiny.Stability <= 0 || !steppedTiny.Due.After(open) {
		t.Fatalf("expected valid positive stability and future Due after stepping tiny stability, got %+v", steppedTiny)
	}
	onTable := review.NewDueByDay(day)
	pBalance := review.Defaults()
	pBalance.IsEvenLoad = true
	// Pre-load the day 3 days out so AssignDueDay shifts a new card's due day if it falls in the window.
	for range 10 {
		onTable.Add(open.AddDate(0, 0, 3))
	}
	simHigh := Simulation{By: NewFSRSAt(0.7), Day: day}
	steppedNewBalanced := simHigh.step(Schedule{}, open, pBalance, onTable)
	if onTable.CountOn(steppedNewBalanced.Due) == 0 {
		t.Fatal("expected step on new card to call AssignDueDay and record in DueByDay")
	}

	// Test getAnswersMap bounds: day < 0, day == 0, day == days-1, day == days.
	simRetains := Simulation{Retains: []int{-1, 0, 2, 3}}
	ansMap := simRetains.getAnswersMap(3)
	if ansMap[-1] || !ansMap[0] || !ansMap[2] || ansMap[3] {
		t.Fatalf("unexpected getAnswersMap(3) for [-1,0,2,3]: %+v", ansMap)
	}

	// Test createProjection, getDaysToLearnGoal, createSimulationDay initial zero fields,
	// startDay resetting observed=0, isDebtNext(!hasDue), isRepeat(!d.hasNew()), stepDebt observed++,
	// stepNew exact minutes boundary and showings=1, finishDay d.today+1 < d.days, finalizeProjection admitted == 0.
	proj := sim.createProjection(5, nil, 2, now, review.Defaults())
	if proj.Learned != 0 || proj.Short != 0 {
		t.Fatalf("expected Learned=0 and Short=0 in createProjection, got %+v", proj)
	}
	if got := sim.getDaysToLearnGoal(review.Defaults(), now); got != 0 {
		t.Fatalf("expected 0 from getDaysToLearnGoal for non-date goal, got %d", got)
	}

	reckoned := newLearnedCount(review.Defaults(), []Schedule{tinyStab}, open)
	sd := sim.createSimulationDay(simulationSetup{
		days: 2, base: 0, open: open, p: review.Defaults(),
		cards: []Schedule{tinyStab}, falls: make([][]int, 2),
		on: review.NewDueByDay(day), reckoned: reckoned, unreviewed: 2,
	})
	if sd.today != 0 || sd.used != 0 || sd.reply != 0 || sd.observed != 0 || sd.begun != 0 ||
		sd.charged != 0 || sd.reviewed != 0 || sd.take != 0 || sd.next != 0 {
		t.Fatalf("expected zero counters on createSimulationDay, got %+v", sd)
	}
	sd.observed = 99
	sd.startDay(0)
	if sd.observed != 0 {
		t.Fatalf("expected startDay to reset observed to 0, got %d", sd.observed)
	}

	// Check isDebtNext and isRepeat when due is present vs new is present, and stepDebt incrementing observed.
	sd.due = []int{0}
	sd.take = 0
	sd.remaining = 1
	sd.admits.Backlog = 50
	if sd.isRepeat() {
		t.Fatal("expected isRepeat() false when due and new cards remain")
	}
	sd.stepDebt()
	if sd.observed != 1 {
		t.Fatalf("expected stepDebt to increment observed to 1, got %d", sd.observed)
	}
	if !sd.isDebtNext() && sd.admits.IsNextCardDebt(sd.observed, sd.begun, sd.isDue(), sd.hasNew()) {
		t.Fatal("expected isDebtNext to match IsNextCardDebt")
	}
	// With 1 debt observed (observed=1, begun=0) and Backlog=50, next card should be new (isDebtNext == false) when hasNew is true!
	sd.due = []int{0}
	sd.take = 0
	if sd.isDebtNext() {
		t.Fatal("expected isDebtNext() false after 1 debt card with 50% backlog and both due and new available")
	}

	// Check stepNew when d.used + d.sim.Cost.New == d.admits.Minutes (exact boundary allowed) vs > d.admits.Minutes,
	// and verify showings appended as 1.
	sd.admits.Limits.Minutes = review.ClosedMinutes
	sd.admits.Minutes = sd.used + sd.sim.Cost.New
	sd.admits.Limits.New = ClosedNothing
	beforeCardsLen := len(sd.cards)
	sd.stepNew()
	if sd.isNewDone || len(sd.cards) != beforeCardsLen+1 || sd.showings[len(sd.showings)-1] != 1 {
		t.Fatalf("expected stepNew to succeed at exact minutes boundary and append showings=1, isNewDone=%v showings=%v",
			sd.isNewDone, sd.showings)
	}
	sd.remaining = 1
	sd.stepNew()
	if !sd.isNewDone || !slices.Contains(sd.closed, review.ClosedMinutes) {
		t.Fatalf("expected stepNew to close with ClosedMinutes when exceeding budget, closed=%v", sd.closed)
	}

	// Check finishDay carrying remaining again cards to d.falls[d.today+1] when d.today+1 < d.days.
	sd.today = 0
	sd.days = 2
	sd.again = []int{0}
	sd.next = 0
	var elapsed time.Duration
	sd.finishDay(&proj, false, &elapsed)
	if !slices.Equal(sd.falls[1], []int{0}) {
		t.Fatalf("expected remaining again card carried to falls[1], got %v", sd.falls[1])
	}

	// Check finalizeProjection when CountAdmitted() == 0.
	projNoAdmit := Projection{Admitted: []bool{false}, Reviewed: []int{0}}
	sim.finalizeProjection(&projNoAdmit, 0, nil, open)
	if math.IsNaN(projNoAdmit.ReviewsADay) || projNoAdmit.ReviewsADay != 0 {
		t.Fatalf("expected 0 ReviewsADay when CountAdmitted()==0, got %g", projNoAdmit.ReviewsADay)
	}
}
