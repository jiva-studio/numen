package indexing

import (
	"context"
	"errors"
	"sync"

	indexdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (p *pass) readNotes(ctx context.Context, read ReadNotes) bool {
	events := p.worker.events
	defer events.FinishTask(work.WalkingNotes)

	events.ReportTask(work.Task{ID: work.WalkingNotes, Title: "Indexing the vault"})
	notes, err := read(ctx, nil)

	switch {
	case err == nil:
	case errors.Is(err, context.Canceled):
		// Stopped on request. The stored data is correct up to that point.
		return false
	default:
		events.ReportError(err)
		return false
	}

	events.ReportNotes(p.v, notes)
	return true
}

func (p *pass) indexSources(ctx context.Context) {
	request := p.isRebuild
	p.isRebuild = false
	p.indexTexts(ctx, request)
}

func (p *pass) embedRecognised(ctx context.Context) {
	target := p.session.drain()
	if len(target) == 0 {
		return
	}
	for path, of := range target {
		p.cutSource(ctx, of, path)
	}
	p.embed(ctx)
}

func (p *pass) indexTexts(ctx context.Context, rebuild bool) {
	events := p.worker.events
	making, err := p.worker.openReading(ctx, p.v, rebuild)
	if err != nil {
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Error: err.Error()})
		return
	}
	texts, cancel := newTaskCancel(ctx)
	defer cancel.stop()
	var current currentSources
	making.OnTextsProgress = func(res indexdomain.IndexResult) {
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
	}

	events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Cancel: cancel.request})
	res, read := making.IndexTexts(texts, p.v)
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
				p.worker.skips.addTexts(path)
			}
		}
		events.FinishTask(work.IndexingTexts)
	default:
		// A failed pass stays in the task list until the user dismisses it.
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, Error: read.Error()})
	}

	p.embed(ctx)
}

func (p *pass) cutSource(ctx context.Context, v domain.Vault, path string) {
	if p.worker.skips.isTextSkipped(path) {
		return
	}
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(work.Task{ID: work.IndexingTexts, Title: taskIndexingTexts, About: path, Error: err.Error()})
	}

	making, err := p.worker.openReading(ctx, v, false)
	if err != nil {
		failed(err)
		return
	}
	switch err := making.CutOne(ctx, v, path); {
	case err == nil:
		// A cut that failed before this one is over.
		events.FinishTask(work.IndexingTexts)
	case !errors.Is(err, context.Canceled):
		failed(err)
	}
}

func (p *pass) embed(ctx context.Context) {
	w := p.worker
	if w.embedder == nil {
		return
	}
	plan := w.planVectors(ctx, p.v)
	if !plan.hasPending {
		w.events.FinishTask(work.MakingVectors)
		p.reportVectorsReady(ctx)
		return
	}

	limit, stop := stopOnWake(ctx, w.read, w.sources, w.notes)
	defer stop()

	if p.makeVectors(ctx, limit, plan) {
		p.reportVectorsReady(ctx)
	}
}

func (p *pass) reportVectorsReady(ctx context.Context) {
	w := p.worker
	if w.vectorsReady == nil || ctx.Err() != nil {
		return
	}
	w.vectorsReady(ctx, p.v)
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
	plan := vectorPlan{generation: w.skips.getGeneration()}
	plan.skipped = w.skips.getSkippedChunks()
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

func (plan vectorPlan) getTotal(res indexdomain.EmbedResult) int64 {
	if plan.pending == 0 {
		return 0
	}
	found := max(0, int64(res.Skipped)-plan.skipped)
	return max(int64(res.Embedded), plan.pending-found)
}

func stopOnWake(ctx context.Context, wakes ...chan struct{}) (context.Context, func()) {
	underCtx, end := context.WithCancel(ctx)
	var watchers sync.WaitGroup
	for _, wake := range wakes {
		watchers.Go(func() {
			select {
			case <-underCtx.Done():
			case <-wake:
				signal(wake)
				end()
			}
		})
	}
	return underCtx, func() {
		end()
		watchers.Wait()
	}
}

func (p *pass) makeVectors(ctx, items context.Context, plan vectorPlan) bool {
	events := p.worker.events
	failed := func(err error) {
		events.ReportTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: err.Error()})
	}

	making, err := p.worker.openReading(ctx, p.v, false)
	if err != nil {
		failed(err)
		return false
	}
	// Progress is reported for the source currently open. The task enters the
	// list when the first source opens, and a progress share is reported once a
	// vector has been created, counted against the total pending.
	items, cancel := newTaskCancel(items)
	defer cancel.stop()
	var current currentSources
	making.OnVectorsProgress = func(res indexdomain.EmbedResult) {
		current.set(getEmbeddingSources(res))
		pos := work.Task{ID: work.MakingVectors, Title: "Indexing", About: res.CurrentSource, Cancel: cancel.request}
		if res.Embedded > 0 {
			pos.Count, pos.Total = int64(res.Embedded), plan.getTotal(res)
		}
		events.ReportTask(pos)
	}

	switch res, err := making.MakeVectors(items, p.v); {
	case err == nil:
		p.worker.skips.setSkippedChunks(res.Skipped, plan.generation)
		events.FinishTask(work.MakingVectors)
		return true
	case errors.Is(err, context.Canceled):
		// The source a person canceled gets no vectors until it changes; the
		// next pass embeds the others.
		if cancel.isRequested.Load() {
			for _, path := range current.get() {
				p.worker.skips.addVectors(path)
			}
		}
		events.FinishTask(work.MakingVectors)
	default:
		// A vault missing vectors is searched by its text alone.
		failed(err)
	}
	return false
}
