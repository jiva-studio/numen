package flashcards

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// countedVersion is the shape of the cache file. A cache of another shape is
// thrown away and worked out again, which costs a reading of the answers.
const countedVersion = 2

type counted struct {
	V int `json:"v"`
	// Runs are the log files this was worked out from, each with the length it
	// had and what it came to. A run is appended to and never rewritten, so a
	// file of the same name and length holds the same answers.
	Runs []countedRun `json:"runs"`
}

type countedRun struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	// Days is what this run alone came to.
	Days map[string]history.Tally `json:"days"`
	// IDs are the identifiers its lines carry, which is what says an answer
	// another run holds too is the one answer.
	IDs []string `json:"ids"`
}

// Reviewed is how much of a vault was answered, and when.
type Reviewed struct {
	// Days is how many answers were given on each day, by the name of the day:
	// the year, the month and the day it began on.
	Days map[string]history.Tally
	// Due is how many card faces fall on each day still to come, by the same
	// names. A card owed today or owed and late is not in it: what is behind is
	// what the front door counts, and this is what is ahead.
	Due map[string]int
	// Retained is how much of what came round in days came back on each day. A
	// card face the scheduler is still putting into memory is not in it.
	Retained map[string]history.Retention
	// Streak is how many days up to now were reviewed without a gap.
	Streak int
	// Answered is how many answers the vault holds altogether.
	Answered int
}

// Counted is how much of a vault was answered on each day it was reviewed.
//
// What a day came to is a sum, and a sum is worked out one file at a time: a
// run arriving from another machine adds to the days it holds and disturbs
// nothing that was counted before it. So the cache is kept by run — a file of
// the name and length it was read at is not read again, and a sitting's own
// file is the only one re-read all evening.
//
// This is what a schedule cannot do. Where an answer leaves a card depends on
// the order of every answer before it, so a schedule arriving late is the whole
// history read again.
type Counted struct {
	Logs port.DerivedStores
	// Kept is where the counting is remembered. A build holding none counts the
	// whole log at every launch.
	Kept port.Schedules
	// Schedules is where the answers have left every card face, which is what
	// says how much falls on each day still to come.
	Schedules Schedules
	Day       history.Day
	Now       func() time.Time
}

// Execute counts one vault.
func (u Counted) Execute(ctx context.Context, v domain.Vault) (Reviewed, error) {
	log := Log{Stores: u.Logs}
	files, err := log.Files(ctx, v)
	if err != nil {
		return Reviewed{}, err
	}

	was := u.remembered(ctx, v)
	now := counted{V: countedVersion}
	out := Reviewed{Days: make(map[string]history.Tally)}

	store, err := u.Logs.Open(v)
	if err != nil {
		return Reviewed{}, err
	}
	// One identifier is one answer over the whole log, so a line another run
	// was counted for is not counted again.
	seen := make(map[string]bool)
	// What is still to come is worked out from the whole history, so every run
	// is read here and the reading is handed on.
	coming := u.Schedules.By != nil
	var held Held
	for _, file := range files {
		var ran Ran
		var opened bool
		one, kept := was[file.Name]
		stale := !kept || one.Size != file.Size
		if stale || coming {
			ran, err = log.Run(ctx, store, file)
			if err != nil {
				return Reviewed{}, err
			}
			opened = true
		}
		if stale {
			one = countedRun{
				Name: file.Name,
				Size: ran.Size,
				Days: history.Counted(u.Day, ran.Answers),
				IDs:  identifiers(ran.Answers),
			}
		}
		now.Runs = append(now.Runs, one)
		held.Skipped += ran.Skipped
		if opened && !ran.Gone && !ran.Shut {
			held.Answers = append(held.Answers, ran.Answers...)
			held.Files = append(held.Files, port.Stored{Name: file.Name, Size: ran.Size})
		}

		// What a run came to on its own is what is kept, and what the run adds
		// to the counting is what no other run has been counted for.
		days := one.Days
		if repeats(one.IDs, seen) {
			if !opened {
				ran, err = log.Run(ctx, store, file)
				if err != nil {
					return Reviewed{}, err
				}
			}
			days = history.Counted(u.Day, given(ran.Answers, seen))
		}
		for _, id := range one.IDs {
			seen[id] = true
		}
		for day, count := range days {
			out.Days[day] = added(out.Days[day], count)
			out.Answered += count.Answered
		}
	}

	held.order = ordered(held.Answers)
	u.remember(ctx, v, now)
	out.Streak = history.Streak(u.Day, out.Days, u.now())

	// What is still to come, and how much came back, are both worked out from
	// the answers in the order they were given, so they are asked for together.
	due, retained, err := u.ahead(ctx, v, held)
	if err != nil {
		return Reviewed{}, err
	}
	out.Due = due
	out.Retained = retained
	return out, nil
}

// added is two days' answers put together, which is how the runs of one day are
// added up: a person may have answered in two sittings, and it is one day.
func added(one, other history.Tally) history.Tally {
	return history.Tally{
		Answered: one.Answered + other.Answered,
		Again:    one.Again + other.Again,
		Hard:     one.Hard + other.Hard,
		Good:     one.Good + other.Good,
		Easy:     one.Easy + other.Easy,
	}
}

// identifiers is what every line of a run is named by, the lines taking an
// answer back among them.
func identifiers(answers []history.Answer) []string {
	out := make([]string, 0, len(answers))
	for _, a := range answers {
		out = append(out, a.ID)
	}
	return out
}

// repeats reports whether a run carries a line another run was counted for.
func repeats(ids []string, seen map[string]bool) bool {
	for _, id := range ids {
		if seen[id] {
			return true
		}
	}
	return false
}

// given is the lines of a run no other run was counted for.
func given(answers []history.Answer, seen map[string]bool) []history.Answer {
	out := make([]history.Answer, 0, len(answers))
	for _, a := range answers {
		if !seen[a.ID] {
			out = append(out, a)
		}
	}
	return out
}

// ahead is how much falls on each day still to come, and how much of what came
// round in days came back on each day behind.
//
// The answers are the reading the days were counted from, so the whole log is
// opened once for the screen.
//
// A card owed today, or owed and late, is not in what is to come: what a person
// owes now is what the front door counts, and this says what is coming after
// it. Where a card falls is worked out from the answers like everything else,
// so the day it shows is the day it would be asked on.
func (u Counted) ahead(
	ctx context.Context, v domain.Vault, held Held,
) (map[string]int, map[string]history.Retention, error) {
	falls := make(map[string]int)
	if u.Schedules.By == nil {
		return falls, nil, nil
	}

	schedules, err := u.Schedules.From(ctx, v, held)
	if err != nil {
		return nil, nil, err
	}

	now := u.now()
	ends := u.Day.Ends(now)
	for _, s := range schedules {
		if !s.Seen() || s.Due.Before(ends) {
			continue
		}
		falls[u.Day.Names(s.Due)]++
	}
	return falls, held.Given().Retained(u.Schedules.By, u.Day), nil
}

// remembered is what was counted last time, by the name of the run it was
// counted from. A cache of another shape is nothing remembered.
func (u Counted) remembered(ctx context.Context, v domain.Vault) map[string]countedRun {
	if u.Kept == nil {
		return nil
	}
	raw, err := u.Kept.Read(ctx, v.ID)
	if err != nil {
		return nil
	}
	var was counted
	if err := json.Unmarshal(raw, &was); err != nil || was.V != countedVersion {
		return nil
	}
	out := make(map[string]countedRun, len(was.Runs))
	for _, one := range was.Runs {
		out[one.Name] = one
	}
	return out
}

// remember puts the counting where the next launch will find it. A cache that
// could not be written is a launch that counts again, so nothing is reported.
func (u Counted) remember(ctx context.Context, v domain.Vault, now counted) {
	if u.Kept == nil {
		return
	}
	raw, err := json.Marshal(now)
	if err != nil {
		return
	}
	_ = u.Kept.Write(ctx, v.ID, raw)
}

func (u Counted) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}
