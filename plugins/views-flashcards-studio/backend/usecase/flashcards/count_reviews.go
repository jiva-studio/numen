package flashcards

import (
	"context"
	"maps"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/cachefile"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// countedVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a reading of the answers.
const countedVersion = 2

// ReviewCounts is how much of a vault was answered, and when.
type ReviewCounts struct {
	Days     map[string]review.Tally
	Due      map[string]int
	Retained map[string]review.RecallTally
	Streak   int
	Answered int
}

// CountReviews is how much of a vault was answered on each day it was reviewed.
// A day's total is a sum, so it is worked out one file at a time: a run arriving from another
// machine adds to the days it holds and disturbs nothing counted before. The cache is kept by
// run, so a file of the name and length it was read at is not read again. A schedule cannot do
// this, since where an answer leaves a card depends on every answer before it.
type CountReviews struct {
	logs      ports.LogStores
	Cache     ports.ScheduleStore
	Report    func(what string, err error)
	schedules Schedules
	day       review.Day
	now       ports.Clock
}

// NewCountReviews is what a vault's days are counted through: where its answers
// are kept, where those answers have left each review card, where one day of
// review gives way to the next, and what time it is.
func NewCountReviews(
	logs ports.LogStores, schedules Schedules, day review.Day, now ports.Clock,
) CountReviews {
	return CountReviews{logs: logs, Cache: nil, Report: nil, schedules: schedules, day: day, now: now}
}

// WithDay returns a copy of CountReviews with the given Day.
func (u CountReviews) WithDay(day review.Day) CountReviews {
	u.day = day
	return u
}

// WithNow returns a copy of CountReviews with the given Clock.
func (u CountReviews) WithNow(now ports.Clock) CountReviews {
	u.now = now
	return u
}

// Execute counts one vault.
func (u CountReviews) Execute(ctx context.Context, v domain.Vault) (ReviewCounts, error) {
	log := Log{Stores: u.logs}
	files, err := log.GetLogFiles(ctx, v)
	if err != nil {
		return ReviewCounts{}, err
	}
	store, err := u.logs.Open(v)
	if err != nil {
		return ReviewCounts{}, err
	}

	run := reviewCounter{
		was: u.readRemembered(ctx, v),
		now: cachefile.Counts{V: countedVersion, Runs: nil},
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Streak is always overwritten from the day tallies before return."
		out:        ReviewCounts{Days: make(map[string]review.Tally), Due: nil, Retained: nil, Streak: 0, Answered: 0},
		countedIDs: make(map[string]bool),
		coming:     u.schedules.scheduler != nil,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="ReviewCounter does not read Skipped from its internal ReviewLog."
		log: ReviewLog{Answers: nil, order: nil, Files: nil, Skipped: 0},
	}
	for _, file := range files {
		if err := u.countFile(ctx, log, store, file, &run); err != nil {
			return ReviewCounts{}, err
		}
	}

	// gomutants:disable-next-line STATEMENT_REMOVE reason="GetHistory computes the same ordered answers when the memoized function is nil."
	run.log.order = newHistory(run.log.Answers)
	u.remember(ctx, v, run.now)
	run.out.Streak = review.Streak(u.day, run.out.Days, u.now.Now())

	// The upcoming cards and the retained cards are both computed from the answers
	// in the order they were given, so they are requested together.
	due, retained, err := u.getDueAndRetained(ctx, v, run.log)
	if err != nil {
		return ReviewCounts{}, err
	}
	run.out.Due = due
	run.out.Retained = retained
	return run.out, nil
}

// reviewCounter is the state of one counting over the runs of a log.
type reviewCounter struct {
	was        map[string]cachefile.Run
	now        cachefile.Counts
	out        ReviewCounts
	countedIDs map[string]bool
	coming     bool
	log        ReviewLog
}

func (u CountReviews) countFile(
	ctx context.Context, log Log, store ports.LogStore, file ports.Entry, run *reviewCounter,
) error {
	var ran LogFile
	var didOpen bool
	var err error
	one, hasCached := run.was[file.Run]
	isStale := !hasCached || one.Size != file.Size
	if isStale || run.coming {
		ran, err = log.ReadFile(ctx, store, file)
		if err != nil {
			return err
		}
		didOpen = true
	}
	if isStale {
		one = cachefile.Run{
			Name: file.Run,
			Size: ran.Size,
			Days: review.GetDayTallies(u.day, ran.Answers),
			IDs:  collectIdentifiers(ran.Answers),
		}
	}
	run.now.Runs = append(run.now.Runs, one)
	if didOpen && !ran.IsGone && !ran.IsUnreadable {
		run.log.Answers = append(run.log.Answers, ran.Answers...)
		run.log.Files = append(run.log.Files, ports.Entry{Run: file.Run, Size: ran.Size})
	}

	if !isRepeated(one.IDs, run.countedIDs) {
		run.addDays(one.IDs, one.Days)
		return nil
	}
	if !didOpen {
		ran, err = log.ReadFile(ctx, store, file)
		if err != nil {
			return err
		}
	}
	run.addDays(one.IDs, review.GetDayTallies(u.day, getUncounted(ran.Answers, run.countedIDs)))
	return nil
}

func (run *reviewCounter) addDays(ids []string, days map[string]review.Tally) {
	for _, id := range ids {
		run.countedIDs[id] = true
	}
	for day, count := range days {
		run.out.Days[day] = addTallies(run.out.Days[day], count)
		run.out.Answered += count.Answered
	}
}

func addTallies(one, other review.Tally) review.Tally {
	return review.Tally{
		Answered: one.Answered + other.Answered,
		Again:    one.Again + other.Again,
		Hard:     one.Hard + other.Hard,
		Good:     one.Good + other.Good,
		Easy:     one.Easy + other.Easy,
	}
}

func collectIdentifiers(answers []review.Answer) []string {
	out := make([]string, 0, len(answers))
	for _, a := range answers {
		out = append(out, a.ID)
	}
	return out
}

func isRepeated(ids []string, countedIDs map[string]bool) bool {
	for _, id := range ids {
		if countedIDs[id] {
			return true
		}
	}
	return false
}

func getUncounted(answers []review.Answer, countedIDs map[string]bool) []review.Answer {
	out := make([]review.Answer, 0, len(answers))
	for _, a := range answers {
		if !countedIDs[a.ID] {
			out = append(out, a)
		}
	}
	return out
}

func (u CountReviews) getDueAndRetained(
	ctx context.Context, v domain.Vault, target ReviewLog,
) (map[string]int, map[string]review.RecallTally, error) {
	falls := make(map[string]int)
	if u.schedules.scheduler == nil {
		return falls, nil, nil
	}

	schedules, err := u.schedules.GetFromLog(ctx, v, target)
	if err != nil {
		return nil, nil, err
	}

	now := u.now.Now()
	ends := u.day.GetEnd(now)
	for _, s := range schedules {
		if s.IsNew() || s.Due.Before(ends) {
			continue
		}
		falls[u.day.GetName(s.Due)]++
	}
	return falls, target.GetHistory().GetRetained(u.schedules.scheduler, u.day), nil
}

func (u CountReviews) readRemembered(ctx context.Context, v domain.Vault) map[string]cachefile.Run {
	if u.Cache == nil {
		return nil
	}
	raw, err := u.Cache.Read(ctx, v.ID)
	if err != nil {
		return nil
	}
	was, err := cachefile.DecodeCounts(raw)
	if err != nil || was.V != countedVersion {
		return nil
	}
	out := make(map[string]cachefile.Run, len(was.Runs))
	for _, one := range was.Runs {
		out[one.Name] = one
	}
	return out
}

func (u CountReviews) remember(ctx context.Context, v domain.Vault, now cachefile.Counts) {
	if u.Cache == nil {
		return
	}
	raw, err := cachefile.EncodeCounts(now)
	// gomutants:disable-next-line BRANCH_IF reason="Marshaling the counts cache with primitive and map fields cannot fail."
	if err != nil {
		return
	}
	// gomutants:disable-next-line CONDITIONALS_NEGATION reason="Cache write failure is reported without affecting the returned counts."
	if err := u.Cache.Write(ctx, v.ID, raw); err != nil && u.Report != nil {
		u.Report("keep the count of reviews of "+string(v.ID), err)
	}
}

// GetReviewDays is the days of a vault that fall inside a stretch.
type GetReviewDays struct {
	counted CountReviews
}

// NewGetReviewDays creates the GetReviewDays scenario.
func NewGetReviewDays(counted CountReviews) GetReviewDays {
	return GetReviewDays{counted: counted}
}

// Execute keeps the days from from to to, both ends inside it; an empty end is unbounded at
// that end. The streak and the answer count are over the whole vault. A day is written with
// each part padded, so it compares as text the way it compares in time.
func (u GetReviewDays) Execute(
	ctx context.Context, v domain.Vault, from, to string,
) (ReviewCounts, error) {
	counts, err := u.counted.Execute(ctx, v)
	if err != nil {
		return ReviewCounts{}, err
	}
	counts.Days = keepDays(counts.Days, from, to)
	counts.Due = keepDays(counts.Due, from, to)
	counts.Retained = keepDays(counts.Retained, from, to)
	return counts, nil
}

// WithDay returns a copy of GetReviewDays with the given Day.
func (u GetReviewDays) WithDay(day review.Day) GetReviewDays {
	u.counted = u.counted.WithDay(day)
	return u
}

// WithNow returns a copy of GetReviewDays with the given Clock.
func (u GetReviewDays) WithNow(now ports.Clock) GetReviewDays {
	u.counted = u.counted.WithNow(now)
	return u
}

func keepDays[T any](days map[string]T, from, to string) map[string]T {
	cloned := maps.Clone(days)
	maps.DeleteFunc(cloned, func(day string, _ T) bool {
		return day < from || (to != "" && day > to)
	})
	return cloned
}
