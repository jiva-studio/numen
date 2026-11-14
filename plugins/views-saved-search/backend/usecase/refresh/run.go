package refresh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

// job is one run of a saved search: the vault, the note, the priority of its calls and the settings.
type job struct {
	vaultID  domain.VaultID
	vault    Vault
	path     string
	priority connection.Priority
	settings Settings
}

func (r *Runner) run(ctx context.Context, one job) (resultcache.Record, error) {
	key := watchKey{vault: one.vaultID, path: one.path}
	r.watchers.begin(key)
	var written resultcache.Record
	defer func() { r.watchers.end(key, written.ComputedAt) }()

	raw, stat, err := one.vault.Notes.Read(ctx, one.path)
	if err != nil {
		written = r.recordFailure(ctx, one, r.findFileName(ctx, one), err)
		return written, err
	}
	want := readTarget(stat, raw, one.settings)
	file := fileKey{vault: one.vaultID, name: resultcache.GetFileName(want.definition.ID, one.path)}
	started := r.marks.get(file)
	if len(want.problems) > 0 {
		written, err = r.finish(ctx, one.vault, file, started, newProblemRecord(one.path, want, r.now()))
		return written, err
	}
	old, hasOld, err := readRecord(ctx, one.vault.Store, file.name)
	if err != nil {
		return written, err
	}
	var reusable *resultcache.Record
	if hasOld {
		reusable = &old
	}
	cached := newCachedReranker(r.reranker, one.priority, reusable, want.queryHash)
	deps := preview.Dependencies{Search: one.vault.Search, Notes: one.vault.Lookup, Reranker: cached}
	input := api.PreviewInput{Query: want.definition.Query, Rerank: want.definition.Rerank, Limit: want.definition.Limit}
	result, err := preview.New(deps, one.settings.Preview).Run(ctx, input)
	switch {
	case err != nil:
		written = r.recordFailure(ctx, one, file.name, err)
		return written, err
	case len(result.Output.Errors) > 0:
		want.problems = moveProblems(result.Output.Errors, raw)
		written, err = r.finish(ctx, one.vault, file, started, newProblemRecord(one.path, want, r.now()))
		return written, err
	case cached.failure != nil:
		written = r.recordFailure(ctx, one, file.name, fmt.Errorf("the reranking service did not answer (%w)", cached.failure))
		return written, nil
	}
	written, err = r.finish(ctx, one.vault, file, started, newRecord(one.path, want, result, cached.scores, r.now()))
	return written, err
}

func (r *Runner) findFileName(ctx context.Context, one job) string {
	refs, err := one.vault.Lookup.GetNotes(ctx, []string{one.path})
	if err != nil {
		return resultcache.GetFileName("", one.path)
	}
	return resultcache.GetFileName(refs[one.path].ID, one.path)
}

func (r *Runner) finish(ctx context.Context, vault Vault, file fileKey, started mark, record resultcache.Record) (resultcache.Record, error) {
	r.records.Lock()
	defer r.records.Unlock()
	if latest := r.marks.get(file); latest.count != started.count {
		record.StaleSince = &latest.at
	}
	if err := writeRecord(ctx, vault.Store, file.name, record); err != nil {
		return resultcache.Record{}, err
	}
	return record, nil
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
		Path: notePath, QueryHash: "", RerankHash: "", Limit: 0, SettingsHash: "", Reranker: nil, Floor: 0,
		ComputedAt: time.Time{}, StaleSince: nil, FailedAt: nil, Failure: "", Order: "", CandidateCount: 0,
		Results: []resultcache.Result{}, RerankScores: map[string]float64{}, Warnings: []string{}, Errors: []api.Problem{},
	}
}

func newProblemRecord(notePath string, want target, now time.Time) resultcache.Record {
	record := newEmptyRecord(notePath)
	record.QueryHash, record.RerankHash = want.queryHash, want.rerankHash
	record.Limit, record.SettingsHash = want.limit, want.settingsHash
	record.ComputedAt, record.Errors = now, want.problems
	return record
}

func newRecord(notePath string, want target, result preview.Run, scores map[string]float64, now time.Time) resultcache.Record {
	record := newEmptyRecord(notePath)
	record.QueryHash, record.RerankHash = want.queryHash, resultcache.HashText(result.Rerank)
	record.Limit, record.SettingsHash = want.limit, want.settingsHash
	record.Floor, record.ComputedAt, record.Order = result.Output.Floor, now, result.Output.Order
	record.CandidateCount = result.Output.CandidateCount
	if result.Reranker != nil {
		record.Reranker = &resultcache.Reranker{ConnectionID: result.Reranker.ConnectionID, Options: result.Reranker.Options}
		record.RerankScores = scores
	}
	for i, one := range result.Output.Results {
		record.Results = append(record.Results, resultcache.Result{PreviewResult: one, ChunkIDs: result.ChunkIDs[i]})
	}
	if result.Output.Warnings != nil {
		record.Warnings = result.Output.Warnings
	}
	return record
}
