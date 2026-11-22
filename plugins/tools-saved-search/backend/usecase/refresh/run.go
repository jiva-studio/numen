package refresh

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/textplace"
)

const (
	searchWait       = 30 * time.Second
	staleDebounce    = 2 * time.Second
	searchPageSize   = 100
	scoreKindRerank  = "rerank"
	failureProblems  = "the query has problems"
	suggestionPrefix = "did you mean"
)

// job is one run of a saved search: the vault, the note, and the priority of its search and whether the
// core shows no task for it.
type job struct {
	vaultID      domain.VaultID
	vault        Vault
	path         string
	priority     host.SearchPriority
	isBackground bool
}

// found is a search that ended: its last state and every page of its results.
type found struct {
	state   host.SearchState
	results []host.SearchResult
}

func (r *Runner) run(ctx context.Context, one job) (resultcache.Record, error) {
	r.watchers.begin(watchKey{vault: one.vaultID, path: one.path})
	return r.runBegun(ctx, one)
}

func (r *Runner) runBegun(ctx context.Context, one job) (resultcache.Record, error) {
	key := watchKey{vault: one.vaultID, path: one.path}
	var written resultcache.Record
	defer func() { r.watchers.end(key, written.ComputedAt) }()

	raw, stat, err := one.vault.Notes.Read(ctx, one.path)
	if err != nil {
		written = r.recordFailure(ctx, one, r.findFileName(ctx, one), err)
		return written, err
	}
	want := readTarget(stat, raw)
	name := resultcache.GetFileName(want.definition.ID, one.path)
	if len(want.problems) > 0 {
		written, err = r.recordProblems(ctx, one, name, want, want.problems)
		return written, err
	}
	start := host.SearchStart{Query: want.definition.Query, Priority: one.priority, IsBackground: one.isBackground}
	result, err := r.runSearch(ctx, one.vault.Search, start)
	switch {
	case err != nil:
		written = r.recordFailure(ctx, one, name, err)
		return written, err
	case len(result.state.Errors) > 0:
		written, err = r.recordProblems(ctx, one, name, want, convertSearchProblems(result.state.Errors))
		return written, err
	}
	written, err = r.write(ctx, one.vault, name, newRecord(one.path, want, result, r.now()))
	return written, err
}

func (r *Runner) runSearch(ctx context.Context, search ports.Search, start host.SearchStart) (found, error) {
	id, err := search.StartSearch(ctx, start)
	if err != nil {
		return found{}, err
	}
	defer func() {
		if err := search.DeleteSearch(context.WithoutCancel(ctx), id); err != nil {
			r.log.Debug("saved search: remove a search", "id", id, "error", err)
		}
	}()
	read := host.SearchRead{Wait: searchWait, PageSize: searchPageSize, PageToken: ""}
	state, err := search.GetSearch(ctx, id, read)
	for err == nil && state.IsRunning {
		state, err = search.GetSearch(ctx, id, read)
	}
	if err != nil {
		return found{}, err
	}
	if len(state.Errors) > 0 {
		return found{state: state, results: nil}, nil
	}
	results := slices.Clone(state.Results)
	for token := state.NextPageToken; token != ""; {
		page, err := search.GetSearch(ctx, id, host.SearchRead{Wait: 0, PageSize: searchPageSize, PageToken: token})
		if err != nil {
			return found{}, err
		}
		results = append(results, page.Results...)
		state.IsStale = state.IsStale || page.IsStale
		token = page.NextPageToken
	}
	return found{state: state, results: results}, nil
}

func (r *Runner) findFileName(ctx context.Context, one job) string {
	refs, err := one.vault.Lookup.GetNotes(ctx, []string{one.path})
	if err != nil {
		return resultcache.GetFileName("", one.path)
	}
	return resultcache.GetFileName(refs[one.path].ID, one.path)
}

func (r *Runner) write(ctx context.Context, vault Vault, name string, record resultcache.Record) (resultcache.Record, error) {
	r.records.Lock()
	defer r.records.Unlock()
	if err := writeRecord(ctx, vault.Store, name, record); err != nil {
		return resultcache.Record{}, err
	}
	return record, nil
}

func (r *Runner) recordProblems(ctx context.Context, one job, name string, want target, problems []resultcache.Problem) (resultcache.Record, error) {
	r.records.Lock()
	defer r.records.Unlock()
	record, hasRecord, err := readRecord(ctx, one.vault.Store, name)
	if err != nil {
		return resultcache.Record{}, err
	}
	if !hasRecord {
		record = newEmptyRecord(one.path)
	}
	now := r.now()
	record.QueryHash, record.Errors = want.queryHash, problems
	record.FailedAt, record.Failure = &now, getFailure(problems)
	if record.StaleSince == nil {
		record.StaleSince = &now
	}
	if err := writeRecord(ctx, one.vault.Store, name, record); err != nil {
		return resultcache.Record{}, err
	}
	return record, nil
}

func getFailure(problems []resultcache.Problem) string {
	messages := make([]string, 0, len(problems))
	for _, problem := range problems {
		if problem.Line > 0 {
			return failureProblems
		}
		messages = append(messages, problem.Message)
	}
	return strings.Join(messages, "; ")
}

func (r *Runner) recordFailure(ctx context.Context, one job, name string, failure error) resultcache.Record {
	if ctx.Err() != nil || errors.Is(failure, host.ErrNotExist) {
		return resultcache.Record{}
	}
	r.records.Lock()
	defer r.records.Unlock()
	record, hasRecord, err := readRecord(ctx, one.vault.Store, name)
	if err != nil {
		return resultcache.Record{}
	}
	if !hasRecord || record.ComputedAt.IsZero() {
		record = newEmptyRecord(one.path)
	}
	now := r.now()
	record.FailedAt, record.Failure = &now, failure.Error()
	if record.StaleSince == nil {
		record.StaleSince = &now
	}
	if err := writeRecord(ctx, one.vault.Store, name, record); err != nil {
		return resultcache.Record{}
	}
	return record
}

func writeRecord(ctx context.Context, store ports.Store, name string, record resultcache.Record) error {
	raw, err := resultcache.Encode(record)
	if err != nil {
		return err
	}
	return store.Write(ctx, name, raw)
}

func newEmptyRecord(notePath string) resultcache.Record {
	return resultcache.Record{
		Path: notePath, QueryHash: "", Floor: 0, ComputedAt: time.Time{}, StaleSince: nil, FailedAt: nil, Failure: "",
		Order: "", Reranker: "", IndexGeneration: "", Results: []resultcache.Result{}, Warnings: []string{}, Errors: []resultcache.Problem{},
	}
}

func newRecord(notePath string, want target, result found, now time.Time) resultcache.Record {
	record := newEmptyRecord(notePath)
	record.QueryHash, record.ComputedAt = want.queryHash, now
	record.Floor, record.IndexGeneration = result.state.Floor, result.state.IndexGeneration
	record.Order = resultcache.OrderFused
	if result.state.ScoreKind == scoreKindRerank {
		record.Order = resultcache.OrderReranked
	}
	if len(result.state.Reranks) > 0 {
		record.Reranker = result.state.Reranks[0].ConnectionName
	}
	if result.state.IsStale {
		record.StaleSince = &now
	}
	for _, one := range result.results {
		record.Results = append(record.Results, newResult(one))
	}
	record.Warnings = append(record.Warnings, result.state.Warnings...)
	return record
}

func newResult(one host.SearchResult) resultcache.Result {
	result := resultcache.Result{
		Path: one.Path, Title: one.Title, Kind: one.Kind, Place: nil, Text: one.Text, Start: one.Start, Length: one.Length,
		Line: one.MatchLine + 1, URI: textplace.FormatNoteAddress(one.Path), Score: one.Score, IsWholeFile: one.IsWholeFile,
	}
	if !one.IsWholeFile {
		result.URI = textplace.FormatAddress(one.Path, one.Start, one.Length)
		result.Place = newPlace(one)
	}
	return result
}

func newPlace(one host.SearchResult) *resultcache.Place {
	s := textplace.GetSelector(one.Selector)
	if s.IsEmpty() {
		return nil
	}
	place := &resultcache.Place{Page: s.Page, PositionSeconds: nil, EndSeconds: nil, CFI: s.CFI, Chapter: ""}
	if s.CFI != "" {
		place.Chapter = one.PlaceLabel
	}
	if len(s.T) > 0 {
		place.PositionSeconds = &s.T[0]
	}
	if len(s.T) > 1 {
		place.EndSeconds = &s.T[1]
	}
	return place
}

func convertSearchProblems(problems []host.SearchProblem) []resultcache.Problem {
	converted := make([]resultcache.Problem, len(problems))
	for i, problem := range problems {
		converted[i] = resultcache.Problem{
			Line: problem.Line, Column: problem.Column, EndLine: problem.EndLine, EndColumn: problem.EndColumn,
			Message: getProblemMessage(problem),
		}
	}
	return converted
}

func getProblemMessage(problem host.SearchProblem) string {
	if problem.Suggestion == "" || strings.Contains(problem.Message, suggestionPrefix) {
		return problem.Message
	}
	return problem.Message + ": " + suggestionPrefix + " " + problem.Suggestion + "?"
}
