package refresh

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/searchnote"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/preview"
)

// target is what a run of a note as it reads now produces a record from: its definition, its problems in
// the positions of the file, and the hashes and the limit the record keeps.
type target struct {
	definition   searchnote.Definition
	problems     []api.Problem
	queryHash    string
	rerankHash   string
	settingsHash string
	limit        int
}

// State is a saved search as the view reads it: the note as it is now, its problems, its cache, and
// whether a run will replace the cache or is on its way.
type State struct {
	Definition searchnote.Definition
	Problems   []api.Problem
	Record     resultcache.Record
	HasRecord  bool
	IsStale    bool
	IsRunning  bool
}

// ReadState reads the note and its cache without a run, a search or a call to the reranker. A note of
// another type returns searchnote.ErrNotASearch.
func (r *Runner) ReadState(ctx context.Context, vault domain.VaultID, notePath string) (State, error) {
	v := r.open(vault)
	raw, stat, err := v.Notes.Read(ctx, notePath)
	if err != nil {
		return State{}, err
	}
	if !searchnote.IsSearch(stat, raw) {
		return State{}, searchnote.ErrNotASearch
	}
	want := readTarget(stat, raw, r.settings())
	record, hasRecord, err := readRecord(ctx, v.Store, resultcache.GetFileName(want.definition.ID, notePath))
	if err != nil {
		return State{}, err
	}
	return State{
		Definition: want.definition, Problems: want.problems, Record: record, HasRecord: hasRecord,
		IsStale:   !hasRecord || record.StaleSince != nil || isChanged(want, record),
		IsRunning: r.IsRunning(vault, notePath),
	}, nil
}

func readTarget(stat domain.FileStat, raw []byte, settings Settings) target {
	definition, problems := searchnote.Read(stat, raw)
	want := target{
		definition: definition, problems: convertProblems(problems), queryHash: resultcache.HashText(definition.Query),
		rerankHash: "", settingsHash: hashSettings(settings.Preview), limit: 0,
	}
	plan, err := preview.New(preview.Dependencies{Search: nil, Notes: nil, Reranker: nil}, settings.Preview).
		Plan(api.PreviewInput{Query: definition.Query, Rerank: definition.Rerank, Limit: definition.Limit})
	if err != nil {
		plan = preview.Plan{Limit: 0, Rerank: definition.Rerank, Problems: nil}
	}
	want.limit, want.rerankHash = plan.Limit, resultcache.HashText(plan.Rerank)
	if len(want.problems) == 0 {
		want.problems = moveProblems(plan.Problems, raw)
	}
	return want
}

func convertProblems(problems []searchnote.Problem) []api.Problem {
	converted := make([]api.Problem, len(problems))
	for i, problem := range problems {
		converted[i] = api.Problem{Line: problem.Line, Column: problem.Column, Message: problem.Message}
	}
	return converted
}

func moveProblems(problems []api.Problem, raw []byte) []api.Problem {
	queryLine, isFound := searchnote.GetQueryLine(raw)
	moved := make([]api.Problem, len(problems))
	for i, problem := range problems {
		if isFound {
			problem.Line += queryLine - 1
		}
		moved[i] = problem
	}
	return moved
}

func hashSettings(config preview.Config) string {
	numbers := []int{
		config.DefaultLimit, config.MaxLimit, config.ResultsPerSearch, config.RerankResultsPerSearch, config.MaxCandidates,
		config.MaxPathPrefixes, config.FusedPerResult, config.MaxFused, config.RerankedPerResult, config.MaxReranked,
		config.MaxRerankedBytes, config.MergeGapBytes, config.MaxQueryBytes,
	}
	texts := make([]string, len(numbers))
	for i, number := range numbers {
		texts[i] = strconv.Itoa(number)
	}
	return resultcache.HashText(strings.Join(texts, ","))
}

func isChanged(want target, record resultcache.Record) bool {
	return want.queryHash != record.QueryHash || want.rerankHash != record.RerankHash || want.limit != record.Limit ||
		want.settingsHash != record.SettingsHash || !slices.Equal(want.problems, record.Errors)
}

// rows are the reranking rows of the setting at the time of a batch; isKnown is false when they could not
// be listed.
type rows struct {
	rerankers []ports.RerankerID
	isKnown   bool
}

func isRerankerChanged(record resultcache.Record, now rows) bool {
	if !now.isKnown || record.ComputedAt.IsZero() || len(record.Errors) > 0 {
		return false
	}
	if record.Reranker != nil {
		return len(now.rerankers) == 0 || !isSameRow(now.rerankers[0], *record.Reranker)
	}
	return record.FailedAt == nil && len(record.Results) > 0 && len(now.rerankers) > 0
}

func isSameRow(row ports.RerankerID, reranker resultcache.Reranker) bool {
	return row.ConnectionID == reranker.ConnectionID && maps.Equal(row.Options, reranker.Options)
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
