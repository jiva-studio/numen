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
	pDate := Defaults()
	pDate.Goal = GoalDate
	pDate.By = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC) // 2 days (Wed, Thu)
	pDate.Load = map[time.Weekday]int{time.Wednesday: 50, time.Thursday: 50}
	allowDate := pDate.GetAllowance(day, now, Spent{New: 1, Reviews: 2, Took: 3 * time.Minute}, 10, 0)
	// reviewDays = 0.5 + 0.5 = 1.0, pace = ceil(10 / 1.0) = 10, Keeps.New = round(0.5 * 10) = 5.
	if allowDate.Keeps.New != 5 || allowDate.New != 4 {
		t.Fatalf("expected Keeps.New=5 and New=4 for GoalDate with 50%% share, got Keeps.New=%d New=%d",
			allowDate.Keeps.New, allowDate.New)
	}
	if allowDate.Limits != (Limits{New: ClosedDate, Reviews: ClosedNothing, Minutes: ClosedNothing, Backlog: ClosedNothing}) {
		t.Fatalf("unexpected GoalDate limits: %+v", allowDate.Limits)
	}
	if allowDate.Minutes != 10*time.Minute-3*time.Minute {
		t.Fatalf("expected 7m remaining minutes, got %v", allowDate.Minutes)
	}

	pRet := Defaults()
	pRet.Goal = GoalRetention
	if got := pRet.getLimits(); got != (Limits{New: ClosedNew, Reviews: ClosedReviews, Minutes: ClosedNothing, Backlog: ClosedNothing}) {
		t.Fatalf("unexpected GoalRetention limits: %+v", got)
	}
	pUnknown := Defaults()
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

	if got := DivideBudget(10, []int{0, 0}); got == nil || !slices.Equal(got, []int{0, 0}) {
		t.Fatalf("expected non-nil [0,0] when total == 0, got %#v", got)
	}
	if got := DivideBudget(10, []int{-1, 0}); got == nil || !slices.Equal(got, []int{0, 0}) {
		t.Fatalf("expected non-nil [0,0] when total < 0, got %#v", got)
	}
	if got := DivideBudget(5, []int{2, 3}); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("expected demands copied when budget == total, got %v", got)
	}

	// Test day.go: Clock with minutes, GetDate zero time components and hour/minute boundary comparisons,
	// boundary(), IsDue with new card whose Due is in the future, GetDayTallies with undo first, GetSpentUnder with 0 and 1ns took.
	if got := Clock(90 * time.Minute); got != "01:30" {
		t.Fatalf("expected 01:30 from Clock(90m), got %q", got)
	}
	d430 := Day{Starts: 4*time.Hour + 30*time.Minute, In: time.UTC}
	if h, m := d430.boundary(); h != 4 || m != 30 {
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
		{ID: "a1", ReviewCard: cMapped, At: now, Rating: Good, Took: -time.Second},
		{ID: "a2", ReviewCard: cMapped, At: now.Add(time.Second), Rating: Good, Took: 1},
	}, map[CardID]string{cMapped: "P.md"}, nil)
	if totals["P.md"].Took != 1 {
		t.Fatalf("expected Took == 1ns (ignoring negative took and counting 1ns), got %v", totals["P.md"].Took)
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
	if a, ok := answer([]byte(`{"v":1,"id":"1","at":"2026-06-10T12:00:00.000Z","card":"","template":"T","r":3}`)); ok || a != (Answer{}) {
		t.Fatalf("expected (zero, false) when Card is empty in answer(), got (%+v, %v)", a, ok)
	}

	// Test schedule.go: newUnansweredSchedule(), recallDecay, Recall with stability <= 0, negative away, and 24h away at stability 1.
	if newUnansweredSchedule() != (Schedule{}) {
		t.Fatalf("expected zero Schedule from newUnansweredSchedule(), got %+v", newUnansweredSchedule())
	}
	if recallDecay != -0.5 {
		t.Fatalf("expected recallDecay == -0.5, got %g", recallDecay)
	}
	if Recall(24*time.Hour, 0) != 0 || Recall(24*time.Hour, -1) != 0 {
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
	pQuietDay := Defaults()
	pQuietDay.IsEvenLoad = true
	// Make the natural due day for a review card (e.g. 3 days after now, Saturday 2026-06-13) have 0 load
	// so AssignDueDay moves it to another day in the window!
	pQuietDay.Load = map[time.Weekday]int{time.Saturday: 0}
	fHighRet := NewFSRSAt(0.97)
	// First answer puts card in learning, second answer puts it in review, third schedules ~3 days out.
	t0 := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	hist := History{
		{ID: "1", ReviewCard: cA, At: t0.Add(-24 * time.Hour), Rating: Good},
		{ID: "2", ReviewCard: cA, At: t0, Rating: Good},
	}
	replayed := hist.Replay(Day{Starts: 0, In: time.UTC}, func(CardID) SchedulingPolicy {
		return SchedulingPolicy{By: fHighRet, Preset: pQuietDay}
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
	if !(Bounds{Min: 0, Max: 10}).Contains(10) || (Bounds{Min: 0, Max: 10}).Contains(10.1) {
		t.Fatal("expected Bounds.Contains inclusive upper bound")
	}
	if NeverClears != -1 || NeverLearns != -1 || LearnsUnasked != -2 {
		t.Fatalf("unexpected sentinel constants: %d, %d, %d", NeverClears, NeverLearns, LearnsUnasked)
	}

	// Test IsLearned at exact retention boundary (Recall == retention) and GetLearnedRule with unknown Rule.
	pRet := Defaults()
	pRet.Rule = RuleRetention
	pRet.Retention = Recall(24*time.Hour, 1.0)
	sExactRet := Schedule{Last: now.Add(-24 * time.Hour), Due: now, Stability: 1.0}
	if !pRet.IsLearned(sExactRet, now) {
		t.Fatal("expected IsLearned true when Recall equals target retention")
	}
	pBadRule := Defaults()
	pBadRule.Rule = "unknown"
	gotRule, _, _ := pBadRule.GetLearnedRule()
	if gotRule != Defaults().Rule {
		t.Fatalf("expected default rule for unknown Rule, got %q", gotRule)
	}

	// Test GetOverallStopReason default Goal case (MinutesADay == 0 vs > 0), GetWeeklyShare, GetStopReason.
	pDefaultGoal := Defaults()
	pDefaultGoal.Goal = "unknown"
	pDefaultGoal.MinutesADay = 0
	if got := pDefaultGoal.GetOverallStopReason(day, now); got != StoppedNoMinutes {
		t.Fatalf("expected StoppedNoMinutes for unknown goal with MinutesADay=0, got %q", got)
	}
	pDefaultGoal.MinutesADay = 1
	if got := pDefaultGoal.GetOverallStopReason(day, now); got != StoppedNothing {
		t.Fatalf("expected StoppedNothing for unknown goal with MinutesADay=1, got %q", got)
	}
	if got := Defaults().GetWeeklyShare(); got != 7 {
		t.Fatalf("expected 7 weekly share for defaults, got %g", got)
	}
	if got := Defaults().GetStopReason(day, now); got != StoppedNothing {
		t.Fatalf("expected StoppedNothing from GetStopReason, got %q", got)
	}

	// Test getPace, getCapacityToBegin, getReviewDays, getDaysOfReview, IsPast across all boundaries.
	pDate := Defaults()
	pDate.Goal = GoalDate
	pDate.By = time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC) // 1 day before now (span == 0)
	if got := pDate.getReviewDays(day, now); got != 0 {
		t.Fatalf("expected 0 review days when By is 1 day before now (span==0), got %g", got)
	}
	if got := pDate.getCapacityToBegin(day, now, 0); got != 0 {
		t.Fatalf("expected 0 capacity when By is 1 day before now (span==0), got %g", got)
	}
	if got := pDate.getPace(day, now, 10, 1); got != 0 {
		t.Fatalf("expected 0 pace when reviewDays == 0, got %d", got)
	}

	pDate.By = time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC) // 1 day after now (span == 2)
	if got := pDate.getReviewDays(day, now); got != 2 {
		t.Fatalf("expected 2 review days when By is 1 day after now, got %g", got)
	}
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

	if got := (Preset{Goal: GoalMinutes, By: now}).getCapacityToBegin(day, now, 1); got != 0 {
		t.Fatalf("expected 0 capacity when Goal != GoalDate, got %g", got)
	}
	if got := (Preset{Goal: GoalDate}).getCapacityToBegin(day, now, 1); got != 0 {
		t.Fatalf("expected 0 capacity when By is zero, got %g", got)
	}
	if got := (Preset{Goal: GoalMinutes, By: now}).getReviewDays(day, now); got != 0 {
		t.Fatalf("expected 0 review days when Goal != GoalDate, got %g", got)
	}
	if got := (Preset{Goal: GoalDate}).getReviewDays(day, now); got != 0 {
		t.Fatalf("expected 0 review days when By is zero, got %g", got)
	}

	// Trigger time.Parse error in getCapacityToBegin and getReviewDays with year 10000.
	year10000 := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := pDate.getCapacityToBegin(day, year10000, 1); got != 0 {
		t.Fatalf("expected 0 capacity when now has 5-digit year, got %g", got)
	}
	if got := pDate.getReviewDays(day, year10000); got != 0 {
		t.Fatalf("expected 0 review days when now has 5-digit year, got %g", got)
	}

	// Test getDaysOfReview across 0, <0, and >7 days with varied weekday shares.
	pVaried := Defaults()
	pVaried.Load = map[time.Weekday]int{
		time.Wednesday: 50,
		time.Thursday:  100,
		time.Friday:    0,
		time.Saturday:  50,
		time.Sunday:    100,
		time.Monday:    0,
		time.Tuesday:   100,
	} // WeeklyShare = 4.0
	if got := pVaried.getDaysOfReview(now, 0); got != 0 {
		t.Fatalf("expected 0 for days=0, got %g", got)
	}
	if got := pVaried.getDaysOfReview(now, -1); got != 0 {
		t.Fatalf("expected 0 for days=-1, got %g", got)
	}
	// 10 days starting Wednesday: 1 whole week (4.0) + Wed (0.5) + Thu (1.0) + Fri (0.0) = 5.5.
	if got := pVaried.getDaysOfReview(now, 10); got != 5.5 {
		t.Fatalf("expected 5.5 days of review for 10 days, got %g", got)
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
	if v, ok := number("not-a-number"); ok || v != 0 {
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
	if got := Defaults().AssignDueDay(nil, now, due3d); !got.Equal(due3d) {
		t.Fatalf("expected due unchanged when DueByDay is nil, got %v", got)
	}
	pNoEven := Defaults()
	pNoEven.IsEvenLoad = false
	table := NewDueByDay(day)
	if got := pNoEven.GetDueDay(table, now, due3d); !got.Equal(due3d) {
		t.Fatalf("expected due unchanged when !ShouldBalanceLoad, got %v", got)
	}
	dueShort := now.Add(24 * time.Hour)
	if got := Defaults().GetDueDay(table, now, dueShort); !got.Equal(dueShort) {
		t.Fatalf("expected due unchanged when window does not open, got %v", got)
	}

	// Test computeDueDay tie-breaking: when all days in [from, to] have equal weight (empty table, default 100% load),
	// the scheduler's chosen dueDay wins the tie because heaviest is initialized to p.getWeight(s, dueDay).
	due4d := now.Add(4 * 24 * time.Hour) // first=3, last=5, dueDay=now+4 (strictly inside [from, to])
	if got := Defaults().GetDueDay(NewDueByDay(day), now, due4d); !got.Equal(due4d) {
		t.Fatalf("expected dueDay to win tie when all days in window have equal weight, got %v want %v", got, due4d)
	}
	// Test boundary dueDay == from (away = 2.5 days -> first=2, last=4, dueDay=now+2 == from)
	// and dueDay == to (away = 2.5 days with due at end of day so dueDay == to).
	dueAtFrom := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC).Add(60 * time.Hour) // pos=00:00, due=+2.5d (day +2 == from)
	if got := Defaults().GetDueDay(NewDueByDay(day), time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC), dueAtFrom); !got.Equal(dueAtFrom) {
		t.Fatalf("expected dueDay == from to win tie, got %v want %v", got, dueAtFrom)
	}
	// When pos is 2026-06-10 20:00 UTC and away is 3.5 days (84h), due is 2026-06-14 08:00 UTC (day +4 == last == to, first=2, last=5 -> wait: days=3.5, slack=1+0.15*1=1.15, first=round(2.35)=2, last=round(4.65)=5).
	// For days=2.5 (60h) and pos=2026-06-10 20:00 UTC, due is 2026-06-13 08:00 UTC (day +3), first=2, last=4.
	// For days=3.0 (72h) and pos=2026-06-10 20:00 UTC, due is 2026-06-13 20:00 UTC (day +3), slack=1.075, first=2, last=4.
	// What if d has Starts: 12*time.Hour, pos is 2026-06-10 11:00 UTC (which is in review date 2026-06-09!),
	// and away is 2.5 days (60h) -> due is 2026-06-12 23:00 UTC (which is in review date 2026-06-12, i.e., posDate + 3 days, whereas last = 4 -> posDate + 4 is 2026-06-13).
	// If away is 6.0 days (144h), slack = 1 + 0.15*3.5 = 1.525, first = 4, last = 8.
	// What if away is 2.6 days (62.4h): slack = 1.015, days-slack = 1.585 -> first=2, days+slack = 3.615 -> last=4.
	// If pos is 2026-06-10 11:00 UTC with Starts=12h (date 2026-06-09) and away is 3.4 days (81.6h):
	// slack = 1 + 0.15*0.9 = 1.135, first = round(2.265) = 2, last = round(4.535) = 5.
	// If away is 2.9 days (69.6h): slack = 1 + 0.15*0.4 = 1.06, first = round(1.84)=2, last = round(3.96)=4.
	// With pos = 2026-06-10 11:00 UTC (Starts=12h -> date 2026-06-09), due = pos + 69.6h = 2026-06-13 08:36 UTC (before 12:00 -> date 2026-06-12 = posDate + 3).
	// With away = 3.0 days (72h) and pos = 2026-06-10 13:00 UTC (Starts=0h -> date 2026-06-10), due = 2026-06-13 13:00 UTC (date +3).
	// How can dueDay == to?
	// Take Starts = 12*time.Hour: pos = 2026-06-10 12:30 UTC (date 2026-06-10), away = 2.9 days (69h36m):
	// Wait: counted.AddDate(0, 0, last) has the same time-of-day as pos (12:30 UTC), so s.GetDayNumber(counted.AddDate(0, 0, last)) is ALWAYS s.GetDayNumber(pos) + last!
	// And when is s.GetDayNumber(due) == s.GetDayNumber(pos) + last?
	// When pos is just before the day boundary (e.g. Starts=0, pos = 2026-06-10 23:00 UTC -> day 2026-06-10)
	// and away is 3.1 days (74.4h):
	// slack = 1 + 0.15*0.6 = 1.09, first = round(2.01) = 2, last = round(4.19) = 4.
	// And due = 2026-06-10 23:00 + 74h24m = 2026-06-14 01:24 UTC -> date 2026-06-14 = 2026-06-10 + 4 = to!
	posLate := time.Date(2026, 6, 10, 23, 0, 0, 0, time.UTC)
	dueAtTo := posLate.Add(74*time.Hour + 24*time.Minute)
	if got := Defaults().GetDueDay(NewDueByDay(day), posLate, dueAtTo); !got.Equal(dueAtTo) {
		t.Fatalf("expected dueDay == to to win tie, got %v want %v", got, dueAtTo)
	}

	// Test getWeight when s.on[day] > 0 vs 0: pre-load dueDay with 1 card so its weight is 1/(1+1)=0.5,
	// and another day has 0 cards (weight 1/(1+0)=1.0) vs 2 cards (weight 1/(1+2)=0.333).
	tableWeighted := NewDueByDay(day)
	tableWeighted.Add(due4d) // dueDay now has 1 card (weight 0.5)
	gotShifted := Defaults().GetDueDay(tableWeighted, now, due4d)
	if gotShifted.Equal(due4d) {
		t.Fatal("expected card shifted away from day that already has 1 card when neighboring days have 0")
	}

	// Test getSlacks and window boundaries: days < 2.5, days == 2.5, days == 7, days == 20, days == 90, days > 90.
	if f0, l0, open0 := window(time.Duration(2.49 * 24 * float64(time.Hour))); open0 || f0 != 0 || l0 != 0 {
		t.Fatalf("expected (0, 0, false) below EvenFrom, got (%d, %d, %v)", f0, l0, open0)
	}
	if f1, l1, open1 := window(time.Duration(2.5 * 24 * float64(time.Hour))); !open1 || f1 != 2 || l1 != 4 {
		t.Fatalf("expected (2, 4, true) at EvenFrom=2.5, got (%d, %d, %v)", f1, l1, open1)
	}
	// At days=10: slack = 1 + 0.15*(7-2.5) + 0.1*(10-7) = 1 + 0.675 + 0.3 = 1.975 -> first=round(8.025)=8, last=round(11.975)=12.
	if f10, l10, open10 := window(10 * 24 * time.Hour); !open10 || f10 != 8 || l10 != 12 {
		t.Fatalf("expected (8, 12, true) at 10 days, got (%d, %d, %v)", f10, l10, open10)
	}
	// At days=50: slack = 1 + 0.15*4.5 + 0.1*13 + 0.05*(50-20) = 1 + 0.675 + 1.3 + 1.5 = 4.475 -> first=round(45.525)=46, last=round(54.475)=54.
	if f50, l50, open50 := window(50 * 24 * time.Hour); !open50 || f50 != 46 || l50 != 54 {
		t.Fatalf("expected (46, 54, true) at 50 days, got (%d, %d, %v)", f50, l50, open50)
	}
}
