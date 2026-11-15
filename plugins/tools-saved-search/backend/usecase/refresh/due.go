package refresh

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
)

// target is what a run of a note as it reads now produces a record from: its definition, the problems of
// the note in the positions of the file, and the hash of its query.
type target struct {
	definition searchnote.Definition
	problems   []resultcache.Problem
	queryHash  string
}

// State is a saved search as the view reads it: the note as it is now, the problems of the note or of
// its last run, its cache, and whether a run will replace the cache or is on its way.
type State struct {
	Definition searchnote.Definition
	Problems   []resultcache.Problem
	Record     resultcache.Record
	HasRecord  bool
	IsStale    bool
	IsRunning  bool
}

// ReadState reads the note and its cache without a run. A note of another type returns
// searchnote.ErrNotASearch.
func (r *Runner) ReadState(ctx context.Context, vault domain.VaultID, notePath string) (State, error) {
	v := r.open(vault)
	raw, stat, err := v.Notes.Read(ctx, notePath)
	if err != nil {
		return State{}, err
	}
	if !searchnote.IsSearch(stat, raw) {
		return State{}, searchnote.ErrNotASearch
	}
	want := readTarget(stat, raw)
	record, hasRecord, err := readRecord(ctx, v.Store, resultcache.GetFileName(want.definition.ID, notePath))
	if err != nil {
		return State{}, err
	}
	problems := want.problems
	if len(problems) == 0 && hasRecord && !isChanged(want, record) {
		problems = record.Errors
	}
	return State{
		Definition: want.definition, Problems: problems, Record: record, HasRecord: hasRecord,
		IsStale:   !hasRecord || record.StaleSince != nil || isChanged(want, record),
		IsRunning: r.IsRunning(vault, notePath),
	}, nil
}

func readTarget(stat domain.FileStat, raw []byte) target {
	definition, problems := searchnote.Read(stat, raw)
	return target{definition: definition, problems: convertProblems(problems), queryHash: resultcache.HashText(definition.Query)}
}

func convertProblems(problems []searchnote.Problem) []resultcache.Problem {
	converted := make([]resultcache.Problem, len(problems))
	for i, problem := range problems {
		converted[i] = resultcache.Problem{Line: problem.Line, Column: problem.Column, Message: problem.Message}
	}
	return converted
}

func moveProblems(problems []resultcache.Problem, raw []byte) []resultcache.Problem {
	queryLine, isFound := searchnote.GetQueryLine(raw)
	moved := make([]resultcache.Problem, len(problems))
	for i, problem := range problems {
		if isFound && problem.Line > 0 {
			problem.Line += queryLine - 1
		}
		moved[i] = problem
	}
	return moved
}

func isChanged(want target, record resultcache.Record) bool {
	return want.queryHash != record.QueryHash
}

func getLastRun(record resultcache.Record) time.Time {
	if record.FailedAt != nil && record.FailedAt.After(record.ComputedAt) {
		return *record.FailedAt
	}
	return record.ComputedAt
}

func getCooldownEnd(record resultcache.Record, settings Settings) time.Time {
	return getLastRun(record).Add(time.Duration(settings.RefreshMinutes) * time.Minute)
}

func readRecord(ctx context.Context, store ports.Store, name string) (resultcache.Record, bool, error) {
	raw, err := store.Read(ctx, name)
	if errors.Is(err, host.ErrNotExist) {
		return resultcache.Record{}, false, nil
	}
	if err != nil {
		return resultcache.Record{}, false, err
	}
	record, ok := resultcache.Decode(raw)
	return record, ok, nil
}
