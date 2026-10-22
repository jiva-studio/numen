package agents

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// EditPreview is how a change is drawn while the call making it is still being written. Nothing
// here is a write: the vault is what changes a note, and this only says what is on its way.
type EditPreview struct {
	// Emit is told each time more of the change has arrived.
	Emit func(ctx context.Context, edit domain.Edit)
	// stretch that stands nowhere or twice is not drawn.
	Locate func(ctx context.Context, path, targetText string) (from, to int, unique bool)
}

// isEnabled reports whether anything can be drawn at all.
func (d EditPreview) isEnabled() bool { return d.Emit != nil && d.Locate != nil }

// draftingRun is a run whose drafts of a change are drawn and never shown as steps.
type draftingRun struct {
	run     port.Run
	steps   chan port.Step
	stopped chan struct{}
	once    sync.Once
}

// newDraftingRun draws the drafts the run reports, and hands on every other step as it comes.
func newDraftingRun(ctx context.Context, run port.Run, preview EditPreview) port.Run {
	d := &draftingRun{
		run:     run,
		steps:   make(chan port.Step, 16),
		stopped: make(chan struct{}),
	}
	go d.forward(ctx, preview)
	return d
}

// GetSteps returns the steps of the run that are not drafts.
func (d *draftingRun) GetSteps() <-chan port.Step { return d.steps }

// Stop ends the run and returns once nothing of it is still running.
func (d *draftingRun) Stop() error {
	d.once.Do(func() { close(d.stopped) })
	return d.run.Stop()
}

// forward hands on the steps until the run is over or stopped.
func (d *draftingRun) forward(ctx context.Context, preview EditPreview) {
	defer close(d.steps)

	previewState := drawing{active: make(map[string]struct{})}
	defer previewState.finish(ctx, preview, "")
	for step := range d.run.GetSteps() {
		if step.Kind == domain.StepDraft {
			previewState.draw(ctx, preview, step)
			continue
		}
		if step.Kind == domain.StepToolDone {
			previewState.finish(ctx, preview, "")
		}
		select {
		case d.steps <- step:
		case <-d.stopped:
			return
		case <-ctx.Done():
			return
		}
	}
}

// drawing remembers where the change being written stands, so that a stretch is looked for once
// for each call.
type drawing struct {
	call, path, targetText string
	from, to          int
	isUnique          bool
	// drawn are the calls a change was drawn for and that are not ended.
	active map[string]struct{}
}

// finish ends the drawing of a call, or of every call where none is named.
func (r *drawing) finish(ctx context.Context, preview EditPreview, call string) {
	if call != "" {
		if _, isDrawn := r.active[call]; isDrawn {
			preview.Emit(ctx, domain.Edit{Change: call, IsDone: true})
			delete(r.active, call)
		}
		return
	}
	for renderedVal := range r.active {
		preview.Emit(ctx, domain.Edit{Change: renderedVal, IsDone: true})
		delete(r.active, renderedVal)
	}
}

// draw says what a draft of a change is, where the note holds the stretch it replaces.
func (r *drawing) draw(ctx context.Context, preview EditPreview, step port.Step) {
	if !preview.isEnabled() || step.Place.Path == "" || step.Match == "" {
		return
	}
	if r.call != step.Call || r.path != step.Place.Path || r.targetText != step.Match {
		r.call, r.path, r.targetText = step.Call, step.Place.Path, step.Match
		r.from, r.to, r.isUnique = preview.Locate(ctx, r.path, r.targetText)
	}
	if !r.isUnique {
		return
	}
	preview.Emit(ctx, domain.Edit{
		Change: step.Call,
		Path:   r.path,
		From:   r.from,
		To:     r.to,
		Text:   step.Text,
	})
	r.active[step.Call] = struct{}{}
}
