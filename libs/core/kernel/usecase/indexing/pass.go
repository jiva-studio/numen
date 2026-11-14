package indexing

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// pass is the indexing of one vault, on the one goroutine that runs it.
type pass struct {
	env       passEnv
	events    Events
	waits     wakeWaits
	skips     *vaultreading.Skips
	session   *Session
	v         domain.Vault
	isRebuild bool
	// isIndexChanged records that a step changed the index of the vault since the last index-changed
	// notification. It carries over the steps that follow at once.
	isIndexChanged bool
	// quietTimer runs from the last note write until the vault has been quiet for the quiet period.
	quietTimer <-chan time.Time
}

// passEnv is what a pass takes from the worker it runs for.
type passEnv interface {
	openReading(ctx context.Context, v domain.Vault, rebuild bool) (ReadWholeVault, error)
	planVectors(ctx context.Context, v domain.Vault) vectorPlan
	hasSource() bool
	waitForQuiet() <-chan time.Time
	notifyIndexChanged(ctx context.Context, v domain.Vault)
}

func (p *pass) run(ctx context.Context, read ReadNotes) {
	// The file texts come after the notes: a vault is useful the moment its
	// notes answer, and a library takes minutes to cut and hours to embed. An index is
	// a cache, so neither has to finish.
	isStopped := false
	if p.readNotes(ctx, read) {
		isStopped = p.indexSources(ctx)
	}
	p.reportIndexChanged(ctx, isStopped)

	// A vault whose first indexing failed still receives writes, so the wait
	// proceeds regardless of the indexing outcome.
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.waits.sourcesWoken():
			p.reportIndexChanged(ctx, p.indexSources(ctx))
		case <-p.waits.readWakeChannel():
			p.reportIndexChanged(ctx, p.embedRecognized(ctx))
		case <-p.waits.notesWoken():
			// The wake is sent only after notes were saved or removed. Every write puts the pass off again.
			p.isIndexChanged = true
			p.quietTimer = p.env.waitForQuiet()
		case <-p.quietTimer:
			p.quietTimer = nil
			p.reportIndexChanged(ctx, p.embed(ctx))
		}
	}
}

// reportIndexChanged notifies that the index of the vault changed, once the steps that changed it leave
// nothing pending: no wake held, no quiet timer running, and no vector step stopped by a wake.
func (p *pass) reportIndexChanged(ctx context.Context, isVectorStepStopped bool) {
	if !p.isIndexChanged || ctx.Err() != nil || isVectorStepStopped || p.quietTimer != nil || p.waits.hasWakes() {
		return
	}
	p.isIndexChanged = false
	p.env.notifyIndexChanged(ctx, p.v)
}

func (p *pass) recordIndexChanged(isChanged bool) {
	p.isIndexChanged = p.isIndexChanged || isChanged
}

func (p *pass) readNotes(ctx context.Context, read ReadNotes) bool {
	events := p.events
	defer events.FinishTask(work.WalkingNotes)

	events.ReportTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})
	res, err := read(ctx, nil)
	// The counts cover what was written before a cancel or a failure.
	p.recordIndexChanged(res.Indexed > 0 || res.Removed > 0)

	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		// Stopped on request. The stored data is correct up to that point.
		return false
	default:
		events.ReportError(err)
		return false
	}

	events.ReportNotes(p.v, res.Notes)
	return true
}

// indexSources cuts the file texts and makes their vectors. It answers whether a wake stopped the vector step.
func (p *pass) indexSources(ctx context.Context) bool {
	request := p.isRebuild
	p.isRebuild = false
	return p.indexTexts(ctx, request)
}

// embedRecognized cuts the sources a recognition extended and makes their vectors. It answers whether a
// wake stopped the vector step.
func (p *pass) embedRecognized(ctx context.Context) bool {
	target := p.session.drain()
	if len(target) == 0 {
		return false
	}
	for path, of := range target {
		p.cutSource(ctx, of, path)
	}
	return p.embed(ctx)
}

func (p *pass) indexTexts(ctx context.Context, rebuild bool) bool {
	events := p.events
	making, err := p.env.openReading(ctx, p.v, rebuild)
	if err != nil {
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Error: err.Error()})
		return false
	}
	texts, cancel := newTaskCancel(ctx)
	defer cancel.stop()
	var current currentSources
	reporting := Reporting{IsTextSkipped: p.skips.IsTextSkipped, OnTextsProgress: func(res vaultreading.IndexResult) {
		if res.CurrentSource != "" {
			current.set([]string{res.CurrentSource})
		}
		events.ReportTask(work.Task{
			ID: work.IndexingTexts, Title: taskIndexingTexts, About: res.CurrentSource,
			// Every file the walk found leaves this pass one of four ways, and all
			// four count as done.
			Count:  int64(res.Extracted + res.Unchanged + res.Unreadable + res.Vanished),
			Total:  int64(res.Seen),
			Cancel: cancel.request,
		})
	}}

	events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Cancel: cancel.request})
	res, read := making.IndexTexts(texts, p.v, reporting)
	p.recordIndexChanged(res.Extracted > 0 || res.Removed > 0)
	switch {
	case read == nil:
		if res.Extracted > 0 {
			events.ReportTexts(p.v, res)
		}
		events.FinishTask(work.IndexingTexts)
	case errors.Is(read, context.Canceled):
		// Stopped on request. The cut data is correct up to that point, and a
		// source a person canceled is left uncut until it changes.
		if cancel.isRequested.Load() {
			for _, path := range current.get() {
				p.skips.AddTexts(path)
			}
		}
		events.FinishTask(work.IndexingTexts)
	default:
		// A failed pass stays in the task list until the user dismisses it.
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Error: read.Error()})
	}

	return p.embed(ctx)
}

func (p *pass) cutSource(ctx context.Context, v domain.Vault, path string) {
	if p.skips.IsTextSkipped(path) {
		return
	}
	events := p.events
	failed := func(err error) {
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, About: path, Error: err.Error()})
	}

	making, err := p.env.openReading(ctx, v, false)
	if err != nil {
		failed(err)
		return
	}
	res, err := making.CutOne(ctx, v, path)
	p.recordIndexChanged(res.Extracted > 0 || res.Removed > 0)
	switch {
	case err == nil:
		// A cut that failed before this one is over.
		events.FinishTask(work.IndexingTexts)
	case !errors.Is(err, context.Canceled):
		failed(err)
	}
}

// embed makes the vectors the chunks of the vault lack. It answers whether a wake stopped the vector step.
func (p *pass) embed(ctx context.Context) bool {
	if !p.env.hasSource() {
		return false
	}
	plan := p.env.planVectors(ctx, p.v)
	if !plan.hasPending {
		p.events.FinishTask(work.MakingVectors)
		return false
	}

	limit, stop := p.waits.stopOnWake(ctx)
	defer stop()

	res, isStopped := p.makeVectors(ctx, limit, plan)
	p.recordIndexChanged(res.Embedded > 0)
	return isStopped
}

// vectorPlan is what a vector pass must create, computed once before the pass starts. Chunks are
// discovered a few hundred at a time, so a total derived during the pass would keep growing.
type vectorPlan struct {
	pending    int64
	hasPending bool
	skipped    int64
	generation uint64
}

func (w *Worker) planVectors(ctx context.Context, v domain.Vault) vectorPlan {
	plan := vectorPlan{generation: w.skips.GetGeneration()}
	plan.skipped = w.skips.GetSkippedChunks()
	if w.vectorCounts == nil {
		plan.hasPending = true
		return plan
	}
	target, embedded, err := w.vectorCounts(ctx, v)
	if err != nil {
		plan.hasPending = true
		return plan
	}
	plan.skipped = min(plan.skipped, max(0, target-embedded))
	plan.pending = max(0, target-embedded-plan.skipped)
	plan.hasPending = plan.pending > 0
	return plan
}

func (plan vectorPlan) getTotal(res vaultreading.EmbedResult) int64 {
	if plan.pending == 0 {
		return 0
	}
	found := max(0, int64(res.Skipped)-plan.skipped)
	return max(int64(res.Embedded), plan.pending-found)
}

// makeVectors runs one vector step. It answers what the step made, and whether a wake stopped it.
func (p *pass) makeVectors(ctx, limit context.Context, plan vectorPlan) (vaultreading.EmbedResult, bool) {
	events := p.events
	failed := func(err error) {
		events.ReportTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := p.env.openReading(ctx, p.v, false)
	if err != nil {
		failed(err)
		return vaultreading.EmbedResult{}, false
	}
	// Progress is reported for the source currently open. The task enters the
	// list when the first source opens, and a progress share is reported once a
	// vector has been created, counted against the total pending.
	items, cancel := newTaskCancel(limit)
	defer cancel.stop()
	var current currentSources
	reporting := Reporting{IsVectorSkipped: p.skips.IsVectorSkipped, OnVectorsProgress: func(res vaultreading.EmbedResult) {
		current.set(getEmbeddingSources(res))
		pos := work.Task{ID: work.MakingVectors, Title: "Indexing", About: res.CurrentSource, Cancel: cancel.request}
		if res.Embedded > 0 {
			pos.Count, pos.Total = int64(res.Embedded), plan.getTotal(res)
		}
		events.ReportTask(pos)
	}}

	res, err := making.MakeVectors(items, p.v, reporting)
	switch {
	case err == nil:
		p.skips.SetSkippedChunks(res.Skipped, plan.generation)
		events.FinishTask(work.MakingVectors)
		return res, false
	case errors.Is(err, context.Canceled):
		// The source a person canceled gets no vectors until it changes; the
		// next pass embeds the others.
		if cancel.isRequested.Load() {
			for _, path := range current.get() {
				p.skips.AddVectors(path)
			}
		}
		events.FinishTask(work.MakingVectors)
		// The step limit ends on a wake or with the session; a person's cancel ends only the step.
		return res, !cancel.isRequested.Load() && limit.Err() != nil && ctx.Err() == nil
	default:
		// A vault missing vectors is searched by its text alone.
		failed(err)
	}
	return res, false
}
