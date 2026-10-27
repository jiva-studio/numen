package flashcards

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// countedVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a reading of the answers.
const countedVersion = 2

type countCache struct {
	V int `json:"v"`
	// Runs are the log files this was worked out from, each with the length it
	// had and what it came to. A run is appended to and never rewritten, so a
	// file of the same name and length holds the same answers.
	Runs []cachedRun `json:"runs"`
}

type cachedRun struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	// Days is what this run alone came to.
	Days map[string]review.Tally `json:"days"`
	// IDs are the identifiers its lines carry, which is what says an answer
	// another run holds too is the one answer.
	IDs []string `json:"ids"`
}

// ReviewCounts is how much of a vault was answered, and when.
type ReviewCounts struct {
	// Days is how many answers were given on each day, by the name of the day:
	// the year, the month and the day it began on.
	Days map[string]review.Tally
	// Due is how many review cards fall on each day still to come, by the same
	// names. A card owed today or owed and late is not in it: what is behind is
	Due map[string]int
	// Retained is how much of what came round in days came back on each day. A
	// review card the scheduler is still putting into memory is not in it.
	Retained map[string]review.RecallTally
	// Streak is how many days up to now were reviewed without a gap.
	Streak int
	// Answered is how many answers the vault holds altogether.
	Answered int
}

// CountReviews is how much of a vault was answered on each day it was reviewed.
//
// What a day came to is a sum, and a sum is worked out one file at a time: a
// run arriving from another machine adds to the days it holds and disturbs
// nothing that was counted before it. So the cache is kept by run — a file of
// the name and length it was read at is not read again, and a session's own
// file is the only one re-read all evening.
//
// This is what a schedule cannot do. Where an answer leaves a card depends on
// the order of every answer before it, so a schedule arriving late is the whole
// history read again.
type CountReviews struct {
	logs ports.LogStores
	// Cache is where the counting is remembered. A build holding none counts the
	// whole log at every launch.
	Cache ports.ScheduleStore
	// schedules is where the answers have left every review card, which is what
	// says how much falls on each day still to come.
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
	return CountReviews{logs: logs, schedules: schedules, day: day, now: now}
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
		was:        u.readRemembered(ctx, v),
		now:        countCache{V: countedVersion},
		out:        ReviewCounts{Days: make(map[string]review.Tally)},
		countedIDs: make(map[string]bool),
		coming:     u.schedules.scheduler != nil,
	}
	for _, file := range files {
		if err := u.countFile(ctx, log, store, file, &run); err != nil {
			return ReviewCounts{}, err
		}
	}

	run.log.order = newHistory(run.log.Answers)
	u.remember(ctx, v, run.now)
	run.out.Streak = review.Streak(u.day, run.out.Days, u.now())

	// What is still to come, and how much came back, are both worked out from
	// the answers in the order they were given, so they are asked for together.
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
	// was is what was counted last time, by the name of the run.
	was map[string]cachedRun
	// now is what this counting remembers.
	now countCache
	out ReviewCounts
	// seen holds every identifier already counted. One identifier is one answer
	// over the whole log, so a line another run was counted for is not counted
	// again.
	countedIDs map[string]bool
	// coming is whether what is still to come is worked out from the whole
	// history, so that every run is read and the reading is handed on.
	coming bool
	log    ReviewLog
}

// countFile adds one run of the log to the counting.
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
		one = cachedRun{
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

	days := one.Days
	if isRepeated(one.IDs, run.countedIDs) {
		if !didOpen {
			ran, err = log.ReadFile(ctx, store, file)
			if err != nil {
				return err
			}
		}
		days = review.GetDayTallies(u.day, getUncounted(ran.Answers, run.countedIDs))
	}
	run.addDays(one.IDs, days)
	return nil
}

// addDays puts the days a run adds to the counting into it and marks the run's
// lines as counted. What a run came to on its own is what is kept, and what
// the run adds is what no other run has been counted for.
func (run *reviewCounter) addDays(ids []string, days map[string]review.Tally) {
	for _, id := range ids {
		run.countedIDs[id] = true
	}
	for day, count := range days {
		run.out.Days[day] = addTallies(run.out.Days[day], count)
		run.out.Answered += count.Answered
	}
}

// addTallies is two days' answers put together, which is how the runs of one
// day are added up: a person may have answered in two sessions, and it is one
// day.
func addTallies(one, other review.Tally) review.Tally {
	return review.Tally{
		Answered: one.Answered + other.Answered,
		Again:    one.Again + other.Again,
		Hard:     one.Hard + other.Hard,
		Good:     one.Good + other.Good,
		Easy:     one.Easy + other.Easy,
	}
}

// collectIdentifiers returns what every line of a run is named by, the lines taking an
// answer back among them.
func collectIdentifiers(answers []review.Answer) []string {
	out := make([]string, 0, len(answers))
	for _, a := range answers {
		out = append(out, a.ID)
	}
	return out
}

// isRepeated reports whether a run carries a line another run was counted for.
func isRepeated(ids []string, countedIDs map[string]bool) bool {
	for _, id := range ids {
		if countedIDs[id] {
			return true
		}
	}
	return false
}

// getUncounted is the lines of a run no other run was counted for.
func getUncounted(answers []review.Answer, countedIDs map[string]bool) []review.Answer {
	out := make([]review.Answer, 0, len(answers))
	for _, a := range answers {
		if !countedIDs[a.ID] {
			out = append(out, a)
		}
	}
	return out
}

// getDueAndRetained is how much falls on each day still to come, and how much
// of what came round in days came back on each day behind.
//
// The answers are the reading the days were counted from, so the whole log is
// opened once for the screen.
//
// A card owed today, or owed and late, is not in what is to come: what a person
// it. Where a card falls is worked out from the answers like everything else,
// so the day it shows is the day it would be asked on.
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

	now := u.now()
	ends := u.day.GetEnd(now)
	for _, s := range schedules {
		if s.IsNew() || s.Due.Before(ends) {
			continue
		}
		falls[u.day.GetName(s.Due)]++
	}
	return falls, target.GetHistory().GetRetained(u.schedules.scheduler, u.day), nil
}

// readRemembered is what was counted last time, by the name of the run it was
// counted from. A cache of another shape is nothing remembered.
func (u CountReviews) readRemembered(ctx context.Context, v domain.Vault) map[string]cachedRun {
	if u.Cache == nil {
		return nil
	}
	raw, err := u.Cache.Read(ctx, v.ID)
	if err != nil {
		return nil
	}
	var was countCache
	if err := json.Unmarshal(raw, &was); err != nil || was.V != countedVersion {
		return nil
	}
	out := make(map[string]cachedRun, len(was.Runs))
	for _, one := range was.Runs {
		out[one.Name] = one
	}
	return out
}

// remember puts the counting where the next launch will find it. A cache that
// could not be written is a launch that counts again, so nothing is reported.
func (u CountReviews) remember(ctx context.Context, v domain.Vault, now countCache) {
	if u.Cache == nil {
		return
	}
	raw, err := json.Marshal(now)
	if err != nil {
		return
	}
	_ = u.Cache.Write(ctx, v.ID, raw)
}

// GetReviewDays is the days of a vault that fall inside a stretch.
type GetReviewDays struct {
	counted CountReviews
}

func NewGetReviewDays(counted CountReviews) GetReviewDays {
	return GetReviewDays{counted: counted}
}

// Execute keeps the days from from to to, both ends inside it; an empty end is
// unbounded at that end. The streak and the answer count are over the whole
// vault.
//
// A day is written as the year, the month and the day, each padded, so it is
// compared as text the way it is compared in time.
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
