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
const countedVersion = 1

type counted struct {
	V int `json:"v"`
	// Runs are the log files this was worked out from, each with the length it
	// had and what it came to. A run is appended to and never rewritten, so a
	// file of the same name and length holds the same answers.
	Runs []countedRun `json:"runs"`
}

type countedRun struct {
	Name string         `json:"name"`
	Size int            `json:"size"`
	Days map[string]int `json:"days"`
}

// Reviewed is how much of a vault was answered, and when.
type Reviewed struct {
	// Days is how many answers were given on each day, by the name of the day:
	// the year, the month and the day it began on.
	Days map[string]int
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
	Day  history.Day
	Now  func() time.Time
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
	out := Reviewed{Days: make(map[string]int)}

	store, err := u.Logs.Open(v)
	if err != nil {
		return Reviewed{}, err
	}
	for _, file := range files {
		one, held := was[file.Name]
		if !held || one.Size != file.Size {
			read, err := log.Run(ctx, store, file)
			if err != nil {
				return Reviewed{}, err
			}
			one = countedRun{
				Name: file.Name,
				Size: read.Size,
				Days: history.Counted(u.Day, read.Answers),
			}
		}
		now.Runs = append(now.Runs, one)
		for day, count := range one.Days {
			out.Days[day] += count
			out.Answered += count
		}
	}

	u.remember(ctx, v, now)
	out.Streak = history.Streak(u.Day, out.Days, u.now())
	return out, nil
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
