package review

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

func TestAllowanceCostAndDayInternals(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) // Wednesday
	day := Day{Starts: 0, In: time.UTC}

	// Test GetAllowance with GoalDate and partial weekday share (50%) so p.GetShare * float64(p.getPace) is tested.
	pDate := NewDefaultPreset()
	pDate.Goal = GoalDate
	pDate.By = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC) // 2 days (Wed, Thu)
	pDate.Load = map[time.Weekday]int{time.Wednesday: 50, time.Thursday: 50}
	allowDate := pDate.GetAllowance(day, now, Spent{New: 1, Reviews: 2, Took: 3 * time.Minute}, 20, 0)
	// reviewDays = 0.5 + 0.5 = 1.0, pace = ceil(20 / 1.0) = 20, Keeps.New = round(0.5 * 20) = 10.
	if allowDate.Keeps.New != 10 || allowDate.New != 9 {
		t.Fatalf("expected Keeps.New=10 and New=9 for GoalDate with 50%% share, got Keeps.New=%d New=%d",
			allowDate.Keeps.New, allowDate.New)
	}
	if allowDate.Limits != (Limits{New: ClosedDate, Reviews: ClosedNothing, Minutes: ClosedNothing, Backlog: ClosedNothing}) {
		t.Fatalf("unexpected GoalDate limits: %+v", allowDate.Limits)
	}
	if allowDate.Minutes != 10*time.Minute-3*time.Minute {
		t.Fatalf("expected 7m remaining minutes, got %v", allowDate.Minutes)
	}

	pRet := NewDefaultPreset()
	pRet.Goal = GoalRetention
	if got := pRet.getLimits(); got != (Limits{New: ClosedNew, Reviews: ClosedReviews, Minutes: ClosedNothing, Backlog: ClosedNothing}) {
		t.Fatalf("unexpected GoalRetention limits: %+v", got)
	}
	pUnknown := NewDefaultPreset()
	pUnknown.Goal = "unknown"
	if got := pUnknown.getLimits(); got != (Limits{New: ClosedNothing, Reviews: ClosedNothing, Minutes: ClosedMinutes, Backlog: ClosedBacklog}) {
		t.Fatalf("unexpected default limits: %+v", got)
	}

	// Test cost.go: LeastAnswers, GetDefaultCost, GetCostUnder skipping unmapped cards, addAnswer took <= 0,
	// getMedian < 10 vs == 10 (even) vs == 11 (odd) and unsorted input, DivideBudget total <= 0 and budget >= total.
	if LeastAnswers != 10 {
		t.Fatalf("expected LeastAnswers == 10, got %d", LeastAnswers)
	}
	if got := GetDefaultCost(); got != (AnswerCost{New: 20 * time.Second, Review: 8 * time.Second}) {
		t.Fatalf("unexpected GetDefaultCost: %+v", got)
	}
	cUnmapped := CardID{Card: "unmapped", Template: "T"}
	cMapped := CardID{Card: "mapped", Template: "T"}
	costs := GetCostUnder(NewFSRS(), []Answer{
		{ID: "1", ReviewCard: cUnmapped, At: now, Rating: Good, Took: 5 * time.Second},
		{ID: "2", ReviewCard: cMapped, At: now, Rating: Good, Took: 0},
	}, map[CardID]string{cMapped: "P.md"})
	if _, hasUnmapped := costs[""]; hasUnmapped || len(costs) != 1 {
		t.Fatalf("expected only mapped preset in GetCostUnder, got %+v", costs)
	}
	if costs["P.md"] != GetDefaultCost() {
		t.Fatalf("expected default cost when only 0-duration answer was recorded, got %+v", costs["P.md"])
	}
	var times answerTimes
	times.addAnswer(false, 1)
	if len(times.begun) != 1 || times.begun[0] != 1 {
		t.Fatalf("expected 1ns answer recorded, got %v", times.begun)
	}

	if med, ok := getMedian(make([]time.Duration, 9)); ok || med != 0 {
		t.Fatalf("expected (0, false) from getMedian with 9 items, got (%v, %v)", med, ok)
	}
	even10 := []time.Duration{
		50 * time.Second, 10 * time.Second, 40 * time.Second, 20 * time.Second, 30 * time.Second,
		6 * time.Second, 2 * time.Second, 8 * time.Second, 4 * time.Second, 12 * time.Second,
	}
	// Sorted: 2s, 4s, 6s, 8s, 10s, 12s, 20s, 30s, 40s, 50s -> middle two are index 4 (10s) and index 5 (12s) -> 11s.
	if med, ok := getMedian(even10); !ok || med != 11*time.Second {
		t.Fatalf("expected 11s median for 10 items, got %v, %v", med, ok)
	}
	odd11 := append(slices.Clone(even10), 60*time.Second)
	// Sorted 11 items -> index 5 is 12s.
	if med, ok := getMedian(odd11); !ok || med != 12*time.Second {
		t.Fatalf("expected 12s median for 11 items, got %v, %v", med, ok)
	}

	if got := DivideBudget(-1, []int{0, 0}); got == nil || !slices.Equal(got, []int{0, 0}) {
		t.Fatalf("expected non-nil [0,0] when total == 0, got %#v", got)
	}
	if got := DivideBudget(-1, []int{-1, 0}); got == nil || !slices.Equal(got, []int{0, 0}) {
		t.Fatalf("expected non-nil [0,0] when total < 0, got %#v", got)
	}
	if got := DivideBudget(5, []int{2, 3}); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("expected demands copied when budget == total, got %v", got)
	}
	const bigDemand = (1 << 53) + 3
	if got := DivideBudget(bigDemand, []int{bigDemand}); !slices.Equal(got, []int{bigDemand}) {
		t.Fatalf("expected exact demand when budget == total for large int, got %v", got)
	}

	// Test day.go: Clock with minutes, GetDate zero time components and hour/minute boundary comparisons,
	// boundary(), IsDue with new card whose Due is in the future, GetDayTallies with undo first, GetSpentUnder with 0 and 1ns took.
	if got := FormatClock(90 * time.Minute); got != "01:30" {
		t.Fatalf("expected 01:30 from Clock(90m), got %q", got)
	}
	d430 := Day{Starts: 4*time.Hour + 30*time.Minute, In: time.UTC}
	if h, m := d430.getBoundary(); h != 4 || m != 30 {
		t.Fatalf("expected boundary (4, 30), got (%d, %d)", h, m)
	}
	dtBefore := d430.GetDate(time.Date(2026, 6, 10, 4, 15, 0, 0, time.UTC))
	if !dtBefore.Equal(time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)) ||
		dtBefore.Hour() != 0 || dtBefore.Minute() != 0 || dtBefore.Second() != 0 || dtBefore.Nanosecond() != 0 {
		t.Fatalf("expected 2026-06-09 00:00:00.0 UTC at 04:15, got %v", dtBefore)
	}
	dtAfterMin := d430.GetDate(time.Date(2026, 6, 10, 4, 45, 0, 0, time.UTC))
	if !dtAfterMin.Equal(time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected 2026-06-10 at 04:45, got %v", dtAfterMin)
	}
	dtNextHourLowMin := d430.GetDate(time.Date(2026, 6, 10, 5, 15, 0, 0, time.UTC))
	if !dtNextHourLowMin.Equal(time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected 2026-06-10 at 05:15 (local.Hour() > h with Minute() < minute), got %v", dtNextHourLowMin)
	}

	if !day.IsDue(Schedule{Last: time.Time{}, Due: now.Add(72 * time.Hour)}, now) {
		t.Fatal("expected IsDue true for new card even if Due is in the future")
	}

	tallies := GetDayTallies(day, []Answer{
		{ID: "u1", At: now, Undoes: "gone"},
		{ID: "a1", ReviewCard: cMapped, At: now, Rating: Good},
	})
	if tallies["2026-06-10"].Answered != 1 {
		t.Fatalf("expected valid answer after undo to be counted in GetDayTallies, got %+v", tallies)
	}

	totals := GetSpentUnder(day, "2026-06-10", []Answer{
		{ID: "a1", ReviewCard: cMapped, At: now, Rating: Good, Took: 0},
		{ID: "a2", ReviewCard: cMapped, At: now.Add(time.Second), Rating: Good, Took: 1},
	}, map[CardID]string{cMapped: "P.md"}, nil)
	if totals["P.md"].Took != 1 {
		t.Fatalf("expected Took == 1ns, got %v", totals["P.md"].Took)
	}
}

func TestFSRSLogAndScheduleInternals(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	f := NewFSRS()
	if len(f.GetName()) != len(FSRSName+".")+8 {
		t.Fatalf("expected 8-hex-char hash suffix in FSRS name, got %q", f.GetName())
	}
	if f.IsSpaced(Schedule{Last: time.Time{}, Phase: uint8(fsrs.Review)}) {
		t.Fatal("expected IsSpaced false when schedule IsNew() even if Phase == Review")
	}
	atNew := f.readAtAnswer(Schedule{}, now)
	if atNew.away != 0 || atNew.last != (Schedule{}) {
		t.Fatalf("expected away=0 and zero last schedule for new card, got %+v", atNew)
	}
	atZeroStab := f.readAtAnswer(Schedule{Last: now.Add(-24 * time.Hour), Phase: uint8(fsrs.Review), Stability: 0}, now)
	if atZeroStab.last.Stability != 0.1 {
		t.Fatalf("expected Review card with Stability<=0 floored to 0.1, got %g", atZeroStab.last.Stability)
	}

	fLowW := NewFSRS()
	fLowW.p.W[0] = 0.05
	nextAgain := fLowW.Next(Schedule{}, now, Again)
	if nextAgain.Stability != 0.1 {
		t.Fatalf("expected getNextFromNew stability floored to 0.1 when W[0]=0.05, got %g", nextAgain.Stability)
	}

	gotRev := f.getNextFromReview(
		atAnswer{
			phase: fsrs.Review,
			away:  5,
			last:  Schedule{Last: now.Add(-5 * 24 * time.Hour), Stability: 5, Difficulty: 5},
			out:   Schedule{Last: now, Reps: 2, Phase: 0},
		},
		now, 0.9, fsrs.Good,
	)
	if gotRev.Phase != uint8(fsrs.Review) {
		t.Fatalf("expected getNextFromReview to set Phase=Review, got %d", gotRev.Phase)
	}
	if clampDifficulty(10.5) != 10 || clampDifficulty(0.5) != 1 {
		t.Fatalf("unexpected clampDifficulty bounds: %g, %g", clampDifficulty(10.5), clampDifficulty(0.5))
	}

	// Test log.go: Write on undo answer (Rating=0, Ms=0 omitted), Read with no newline (empty vs non-empty),
	// Read with invalid line before valid line, newNoAnswer(), answer() with empty Card.
	undoRaw, err := Write(Answer{ID: "u1", At: now, Undoes: "a1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(undoRaw), `"r":`) || strings.Contains(string(undoRaw), `"ms":`) {
		t.Fatalf("expected zero Rating and Ms omitted on undo line, got %s", string(undoRaw))
	}
	if ans, skipped := Read([]byte("   ")); ans != nil || skipped != 0 {
		t.Fatalf("expected (nil, 0) for whitespace without newline, got (%v, %d)", ans, skipped)
	}
	if ans, skipped := Read([]byte("\n   ")); ans != nil || skipped != 1 {
		t.Fatalf("expected (nil, 1) for blank line before trailing whitespace, got (%v, %d)", ans, skipped)
	}
	if ans, skipped := Read([]byte("partial-line")); ans != nil || skipped != 1 {
		t.Fatalf("expected (nil, 1) for non-empty without newline, got (%v, %d)", ans, skipped)
	}
	if ans, skipped := Read([]byte{}); ans != nil || skipped != 0 {
		t.Fatalf("expected (nil, 0) for empty input, got (%v, %d)", ans, skipped)
	}
	validRaw, err := Write(Answer{
		ID:         "a1",
		ReviewCard: CardID{Card: "c1", Template: "T"},
		At:         now,
		Rating:     Good,
		Took:       time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ans, skipped := Read(validRaw[:len(validRaw)-1]); len(ans) != 0 || skipped != 1 {
		t.Fatalf("expected (0, 1) when trailing newline is missing on valid JSON, got len=%d skipped=%d", len(ans), skipped)
	}
	if ans, skipped := Read(append([]byte("\n"), validRaw...)); len(ans) != 1 || skipped != 0 {
		t.Fatalf("expected empty line skipped without incrementing skipped count, got len=%d skipped=%d", len(ans), skipped)
	}
	gotAns, gotSkip := Read(append([]byte("bad-json\n"), validRaw...))
	if len(gotAns) != 1 || gotSkip != 1 {
		t.Fatalf("expected 1 valid answer and 1 skipped after invalid line, got len=%d skipped=%d", len(gotAns), gotSkip)
	}
	if newNoAnswer() != (Answer{}) {
		t.Fatalf("expected zero Answer from newNoAnswer(), got %+v", newNoAnswer())
	}
	if a, ok := answer([]byte("{bad")); ok || a != (Answer{}) {
		t.Fatalf("expected (zero, false) on bad JSON in answer(), got (%+v, %v)", a, ok)
	}
	if a, ok := answer([]byte(`{"v":1,"id":"1","at":"2026-06-10T12:00:00.000Z","undo":"u1",bad}`)); ok || a != (Answer{}) {
		t.Fatalf("expected (zero, false) on malformed JSON containing undo in answer(), got (%+v, %v)", a, ok)
	}
	if a, ok := answer([]byte(`{"v":1,"id":"1","at":"2026-06-10T12:00:00.000Z","card":"","template":"T","rating":3}`)); ok || a != (Answer{}) {
		t.Fatalf("expected (zero, false) when Card is empty in answer(), got (%+v, %v)", a, ok)
	}

	// Test schedule.go: newUnansweredSchedule(), recallDecay, Recall with stability <= 0, negative away, and 24h away at stability 1.
	if newUnansweredSchedule() != (Schedule{}) {
		t.Fatalf("expected zero Schedule from newUnansweredSchedule(), got %+v", newUnansweredSchedule())
	}
	if recallDecay != -0.5 {
		t.Fatalf("expected recallDecay == -0.5, got %g", recallDecay)
	}
	if Recall(0, 0) != 0 || Recall(24*time.Hour, -0.5) != 0 {
		t.Fatal("expected Recall == 0 for stability <= 0")
	}
	if Recall(-24*time.Hour, 1) != 1 {
		t.Fatalf("expected Recall == 1 when away < 0 (clamped to 0 days), got %g", Recall(-24*time.Hour, 1))
	}
	if math.Abs(Recall(24*time.Hour, 1)-0.9) > 1e-12 {
		t.Fatalf("expected Recall(24h, 1.0) == 0.9, got %.15f", Recall(24*time.Hour, 1))
	}

	// Test replay.go: Replay calling AssignDueDay, and walkAnswers returning non-nil map on empty history.
	if walked := (History(nil)).walkAnswers(f, nil); walked == nil || len(walked) != 0 {
		t.Fatalf("expected non-nil empty map from walkAnswers, got %#v", walked)
	}
	cA := CardID{Card: "cA", Template: "T"}
	cB := CardID{Card: "cB", Template: "T"}
	pQuietDay := NewDefaultPreset()
	pQuietDay.IsEvenLoad = true
	// Make the natural due day for a review card (3 days after now, Saturday 2026-06-13) have 0 load
	// so AssignDueDay moves it to another day in the window.
	pQuietDay.Load = map[time.Weekday]int{time.Saturday: 0}
	t0 := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	hist := History{
		{ID: "1", ReviewCard: cA, At: t0.Add(-24 * time.Hour), Rating: Good},
		{ID: "2", ReviewCard: cA, At: t0, Rating: Good},
	}
	replayed := hist.Replay(Day{Starts: 0, In: time.UTC}, func(CardID) SchedulingPolicy {
		return SchedulingPolicy{By: f, Preset: pQuietDay}
	})
	if replayed[cA].Due.Weekday() == time.Saturday {
		t.Fatalf("expected Replay to apply AssignDueDay and avoid zero-load Saturday, got %v", replayed[cA].Due)
	}
	_ = cB
}

func TestPresetAndSchedulingInternals(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC) // Wednesday
	day := Day{Starts: 0, In: time.UTC}

	if GetMinutesADayBounds() != (Bounds{Min: 0, Max: 24 * 60}) {
		t.Fatalf("unexpected GetMinutesADayBounds: %+v", GetMinutesADayBounds())
	}
	if GetNewADayBounds() != (Bounds{Min: 0, Max: 9999}) {
		t.Fatalf("unexpected GetNewADayBounds: %+v", GetNewADayBounds())
	}
	if GetReviewsADayBounds() != (Bounds{Min: 0, Max: 9999}) {
		t.Fatalf("unexpected GetReviewsADayBounds: %+v", GetReviewsADayBounds())
	}
	if GetIntervalBounds() != (Bounds{Min: 1, Max: 365}) {
		t.Fatalf("unexpected GetIntervalBounds: %+v", GetIntervalBounds())
	}
	if GetBacklogBounds() != (Bounds{Min: 0, Max: 100}) {
		t.Fatalf("unexpected GetBacklogBounds: %+v", GetBacklogBounds())
	}
	if GetLoadBounds() != (Bounds{Min: 0, Max: FullLoad}) {
		t.Fatalf("unexpected GetLoadBounds: %+v", GetLoadBounds())
	}
	if !(Bounds{Min: 0, Max: 10}).IsWithin(10) || (Bounds{Min: 0, Max: 10}).IsWithin(10.1) {
		t.Fatal("expected Bounds.Contains inclusive upper bound")
	}
	if NeverClears != -1 || NeverLearns != -1 || LearnsUnasked != -2 {
		t.Fatalf("unexpected sentinel constants: %d, %d, %d", NeverClears, NeverLearns, LearnsUnasked)
	}

	// Test IsLearned at exact retention boundary (Recall == retention) and GetLearnedRule with unknown Rule.
	pRet := NewDefaultPreset()
	pRet.Rule = RuleRetention
	pRet.Retention = Recall(24*time.Hour, 1.0)
	sExactRet := Schedule{Last: now.Add(-24 * time.Hour), Due: now, Stability: 1.0}
	if !pRet.IsLearned(sExactRet, now) {
		t.Fatal("expected IsLearned true when Recall equals target retention")
	}
	pBadRule := NewDefaultPreset()
	pBadRule.Rule = "unknown"
	gotRule, _, _ := pBadRule.GetLearnedRule()
	if gotRule != NewDefaultPreset().Rule {
		t.Fatalf("expected default rule for unknown Rule, got %q", gotRule)
	}

	// Test GetOverallStopReason default Goal case (MinutesADay == 0 vs > 0), GetWeeklyShare, GetStopReason.
	pDefaultGoal := NewDefaultPreset()
	pDefaultGoal.Goal = "unknown"
	pDefaultGoal.MinutesADay = 0
	if got := pDefaultGoal.GetOverallStopReason(day, now); got != StoppedNoMinutes {
		t.Fatalf("expected StoppedNoMinutes for unknown goal with MinutesADay=0, got %q", got)
	}
	pDefaultGoal.MinutesADay = 1
	if got := pDefaultGoal.GetOverallStopReason(day, now); got != StoppedNothing {
		t.Fatalf("expected StoppedNothing for unknown goal with MinutesADay=1, got %q", got)
	}
	if got := NewDefaultPreset().GetWeeklyShare(); got != 7 {
		t.Fatalf("expected 7 weekly share for defaults, got %g", got)
	}
	if got := NewDefaultPreset().GetStopReason(day, now); got != StoppedNothing {
		t.Fatalf("expected StoppedNothing from GetStopReason, got %q", got)
	}
	pNoMin := NewDefaultPreset()
	pNoMin.MinutesADay = 0
	if got := pNoMin.GetStopReason(day, now); got != StoppedNoMinutes {
		t.Fatalf("expected StoppedNoMinutes from GetStopReason, got %q", got)
	}

	// Test getPace, getCapacityToBegin, IsPast across all boundaries.
	pDate := NewDefaultPreset()
	pDate.Goal = GoalDate
	pDate.By = time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC) // 1 day before now (span == 0)
	if got := pDate.getCapacityToBegin(day, now, 0); got != 0 {
		t.Fatalf("expected 0 capacity when By is 1 day before now (span==0), got %g", got)
	}
	if got := pDate.getPace(day, now, 10, 1); got != 0 {
		t.Fatalf("expected 0 pace when reviewDays == 0, got %d", got)
	}

	pDate.By = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC) // 1 day after now (span == 2)
	if got := pDate.getCapacityToBegin(day, now, 0); got != 2 {
		t.Fatalf("expected 2 capacity when By is 1 day after now and daysToLearn=0, got %g", got)
	}
	if got := pDate.getCapacityToBegin(day, now, 1); got != 1 {
		t.Fatalf("expected 1 capacity when By is 1 day after now and daysToLearn=1, got %g", got)
	}
	if got := pDate.getCapacityToBegin(day, now, 2); got != 0 {
		t.Fatalf("expected 0 capacity when By is 1 day after now and daysToLearn=2, got %g", got)
	}
	if got := pDate.getPace(day, now, 10, NeverLearns); got != 10 {
		t.Fatalf("expected unbegunCards (10) when daysToLearn == NeverLearns, got %d", got)
	}
	if got := pDate.getPace(day, now, 10, 2); got != 10 {
		t.Fatalf("expected unbegunCards (10) when capacity in == 0, got %d", got)
	}
	if got := pDate.getPace(day, now, 10, 0); got != 5 {
		t.Fatalf("expected ceil(10/2)=5 when capacity in == 2, got %d", got)
	}
	pHalfCap := pDate
	pHalfCap.Load = map[time.Weekday]int{time.Wednesday: 50, time.Thursday: 0}
	if got := pHalfCap.getPace(day, now, 10, 0); got != 20 {
		t.Fatalf("expected ceil(10/0.5)=20 when capacity in == 0.5, got %d", got)
	}

	if got := (Preset{Goal: GoalMinutes, By: now}).getCapacityToBegin(day, now, 0); got != 0 {
		t.Fatalf("expected 0 capacity when Goal != GoalDate, got %g", got)
	}
	if got := (Preset{Goal: GoalDate}).getCapacityToBegin(day, time.Time{}, 0); got != 0 {
		t.Fatalf("expected 0 capacity when By is zero, got %g", got)
	}

	// Trigger time.Parse error in getCapacityToBegin with year 10000.
	year10000 := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := pDate.getCapacityToBegin(day, year10000, 0); got != 0 {
		t.Fatalf("expected 0 capacity when now has 5-digit year, got %g", got)
	}

	// Test getCapacityToBegin across >20 days and single-weekday shares.
	pLong := NewDefaultPreset()
	pLong.Goal = GoalDate
	pLong.By = time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC) // 23 days after 2026-06-10 -> 24 days inclusive
	if got := pLong.getCapacityToBegin(day, now, 0); got != 24 {
		t.Fatalf("expected 24 capacity for 24-day span, got %g", got)
	}
	pThuOnly := NewDefaultPreset()
	pThuOnly.Goal = GoalDate
	pThuOnly.By = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC) // Wed + Thu
	pThuOnly.Load = map[time.Weekday]int{
		time.Wednesday: 0,
		time.Thursday:  100,
		time.Friday:    0,
		time.Saturday:  0,
		time.Sunday:    0,
		time.Monday:    0,
		time.Tuesday:   0,
	}
	if got := pThuOnly.getCapacityToBegin(day, now, 0); got != 1 {
		t.Fatalf("expected 1 capacity for Wed(0)+Thu(100) with daysToLearn=0, got %g", got)
	}
	if got := pThuOnly.getCapacityToBegin(day, now, 1); got != 0 {
		t.Fatalf("expected 0 capacity for Wed(0)+Thu(100) with daysToLearn=1 (only Wed can begin), got %g", got)
	}

	if (Preset{Goal: GoalMinutes, By: now.Add(-48 * time.Hour)}).IsPast(day, now) {
		t.Fatal("expected IsPast false when Goal != GoalDate")
	}
	if (Preset{Goal: GoalDate}).IsPast(day, now) {
		t.Fatal("expected IsPast false when By is zero")
	}

	// Test ReadPreset with explicit nil values in frontmatter map and even_load: false.
	gotPreset, probs := ReadPreset(map[string]any{
		"goal":          nil,
		"learned":       nil,
		"counts":        nil,
		"by_date":       nil,
		"retention":     nil,
		"even_load":     nil,
		"load":          nil,
		"minutes_a_day": nil,
	})
	if len(probs) != 0 || !gotPreset.IsEvenLoad {
		t.Fatalf("expected no problems and default preset for nil map entries, got %+v, probs=%v", gotPreset, probs)
	}
	gotEvenFalse, probs := ReadPreset(map[string]any{"even_load": false})
	if len(probs) != 0 || gotEvenFalse.IsEvenLoad {
		t.Fatalf("expected IsEvenLoad=false when even_load: false, got %+v, probs=%v", gotEvenFalse, probs)
	}
	_, probsDateTimeZero := ReadPreset(map[string]any{"goal": "by_date", "by_date": time.Time{}})
	if len(probsDateTimeZero) != 1 {
		t.Fatalf("expected problem when goal is by_date and by_date is zero time.Time, got %v", probsDateTimeZero)
	}
	if v, ok := parseNumber("not-a-number"); ok || v != 0 {
		t.Fatalf("expected (0, false) from number(string), got (%g, %v)", v, ok)
	}

	// Test scheduling.go: EvenFrom, CountOn(nil), computeDueDay branches (nil DueByDay, !ShouldBalanceLoad, !opens,
	// tie-breaking preferring dueDay inside [from, to] vs dueDay outside [from, to], weight calculation),
	// getSlacks, and window boundaries.
	if EvenFrom != 2.5 || EvenTo != 90 {
		t.Fatalf("unexpected EvenFrom/EvenTo: %g, %d", EvenFrom, EvenTo)
	}
	var nilDue *DueByDay
	if nilDue.CountOn(now) != 0 {
		t.Fatalf("expected 0 from nil DueByDay.CountOn")
	}
	due3d := now.Add(3 * 24 * time.Hour)
	if got := NewDefaultPreset().AssignDueDay(nil, now, due3d); !got.Equal(due3d) {
		t.Fatalf("expected due unchanged when DueByDay is nil, got %v", got)
	}
	pNoEven := NewDefaultPreset()
	pNoEven.IsEvenLoad = false
	table := NewDueByDay(day)
	if got := pNoEven.GetDueDay(table, now, due3d); !got.Equal(due3d) {
		t.Fatalf("expected due unchanged when !ShouldBalanceLoad, got %v", got)
	}
	dueShort := now.Add(24 * time.Hour)
	if got := NewDefaultPreset().GetDueDay(table, now, dueShort); !got.Equal(dueShort) {
		t.Fatalf("expected due unchanged when window does not open, got %v", got)
	}

	// Test computeDueDay tie-breaking: when all days in [from, to] have equal weight (empty table, default 100% load),
	// the scheduler's chosen dueDay wins the tie because heaviest is initialized to p.getWeight(s, dueDay).
	due4d := now.Add(4 * 24 * time.Hour) // first=3, last=5, dueDay=now+4 (strictly inside [from, to])
	if got := NewDefaultPreset().GetDueDay(NewDueByDay(day), now, due4d); !got.Equal(due4d) {
		t.Fatalf("expected dueDay to win tie when all days in window have equal weight, got %v want %v", got, due4d)
	}
	// Test boundary dueDay == from (away = 2.5 days -> first=2, last=4, dueDay=now+2 == from)
	// and dueDay == to (away = 2.5 days with due at end of day so dueDay == to).
	dueAtFrom := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC).Add(60 * time.Hour) // pos=00:00, due=+2.5d (day +2 == from)
	if got := NewDefaultPreset().GetDueDay(NewDueByDay(day), time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), dueAtFrom); !got.Equal(dueAtFrom) {
		t.Fatalf("expected dueDay == from to win tie, got %v want %v", got, dueAtFrom)
	}
	// At 23:00 UTC + 3.1d (74h24m), due lands on day +4 == to while last == 4.
	posLate := time.Date(2026, 6, 10, 23, 0, 0, 0, time.UTC)
	dueAtTo := posLate.Add(74*time.Hour + 24*time.Minute)
	if got := NewDefaultPreset().GetDueDay(NewDueByDay(day), posLate, dueAtTo); !got.Equal(dueAtTo) {
		t.Fatalf("expected dueDay == to to win tie, got %v want %v", got, dueAtTo)
	}

	// Test getWeight when s.on[day] > 0 vs 0: pre-load dueDay with 1 card so its weight is 1/(1+1)=0.5,
	// and another day has 0 cards (weight 1/(1+0)=1.0) vs 2 cards (weight 1/(1+2)=0.333).
	tableWeighted := NewDueByDay(day)
	tableWeighted.Add(due4d) // dueDay now has 1 card (weight 0.5)
	if got := pNoEven.GetDueDay(tableWeighted, now, due4d); !got.Equal(due4d) {
		t.Fatalf("expected due unchanged when !ShouldBalanceLoad even on weighted table, got %v", got)
	}
	gotShifted := NewDefaultPreset().GetDueDay(tableWeighted, now, due4d)
	if gotShifted.Equal(due4d) {
		t.Fatal("expected card shifted away from day that already has 1 card when neighboring days have 0")
	}
	// Test window bounds [from, to]: when days inside [from, to] (days +3, +5) have 2 cards and dueDay (+4) has 1 card,
	// while days outside [from, to] (+2, +6) have 0 cards, dueDay wins and is not moved outside [from, to].
	tableBounded := NewDueByDay(day)
	tableBounded.Add(now.Add(3 * 24 * time.Hour))
	tableBounded.Add(now.Add(3 * 24 * time.Hour))
	tableBounded.Add(due4d)
	tableBounded.Add(now.Add(5 * 24 * time.Hour))
	tableBounded.Add(now.Add(5 * 24 * time.Hour))
	if got := NewDefaultPreset().GetDueDay(tableBounded, now, due4d); !got.Equal(due4d) {
		t.Fatalf("expected dueDay to win inside [from, to] when days outside have 0 cards, got %v want %v", got, due4d)
	}
	// Test exact division in getWeight: dueDay (Sunday = now+4d) has Load=50 and 0 cards -> weight 0.5/(1+0) = 0.5,
	// while Saturday (now+3d) and Monday (now+5d) have Load=100 and 1 card -> weight 1.0/(1+1) = 0.5.
	// With division, all weights equal 0.5 and dueDay wins the tie; with multiplication, 1.0*2 = 2.0 > 0.5*1 and dueDay loses.
	pTieWeight := NewDefaultPreset()
	pTieWeight.Load = map[time.Weekday]int{time.Sunday: 50}
	tableTie := NewDueByDay(day)
	tableTie.Add(now.Add(3 * 24 * time.Hour))
	tableTie.Add(now.Add(5 * 24 * time.Hour))
	if got := pTieWeight.GetDueDay(tableTie, now, due4d); !got.Equal(due4d) {
		t.Fatalf("expected dueDay with 50%% load and 0 cards (0.5/1) to tie 100%% load and 1 card (1.0/2), got %v want %v", got, due4d)
	}

	if !slices.Equal(getSlacks(), []slack{{2.5, 7, 0.15}, {7, 20, 0.1}, {20, math.Inf(1), 0.05}}) {
		t.Fatalf("unexpected getSlacks: %+v", getSlacks())
	}
	// Test getSlacks and window boundaries: days < 2.5, days == 2.5, days == 7, days == 20, days == 90, days > 90.
	if f0, l0, open0 := computeWindow(time.Duration(2.49 * 24 * float64(time.Hour))); open0 || f0 != 0 || l0 != 0 {
		t.Fatalf("expected (0, 0, false) below EvenFrom, got (%d, %d, %v)", f0, l0, open0)
	}
	if f1, l1, open1 := computeWindow(time.Duration(2.5 * 24 * float64(time.Hour))); !open1 || f1 != 2 || l1 != 4 {
		t.Fatalf("expected (2, 4, true) at EvenFrom=2.5, got (%d, %d, %v)", f1, l1, open1)
	}
	// At days=10: slack = 1 + 0.15*(7-2.5) + 0.1*(10-7) = 1 + 0.675 + 0.3 = 1.975 -> first=round(8.025)=8, last=round(11.975)=12.
	if f10, l10, open10 := computeWindow(10 * 24 * time.Hour); !open10 || f10 != 8 || l10 != 12 {
		t.Fatalf("expected (8, 12, true) at 10 days, got (%d, %d, %v)", f10, l10, open10)
	}
	// At days=50: slack = 1 + 0.15*4.5 + 0.1*13 + 0.05*(50-20) = 1 + 0.675 + 1.3 + 1.5 = 4.475 -> first=round(45.525)=46, last=round(54.475)=54.
	if f50, l50, open50 := computeWindow(50 * 24 * time.Hour); !open50 || f50 != 46 || l50 != 54 {
		t.Fatalf("expected (46, 54, true) at 50 days, got (%d, %d, %v)", f50, l50, open50)
	}
}
