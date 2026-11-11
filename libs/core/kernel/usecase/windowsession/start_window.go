package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// modelRecorder keeps the model the vault is read under.
type modelRecorder interface {
	RecordModel(name, recipe string)
}

// rebuilder builds the passes behind the open vault again.
type rebuilder interface {
	Execute(ctx context.Context)
}

// ModelFollow records the model of the embedder when it changes, wakes the indexing of the sources and
// builds the passes again when the size of a chunk changed.
type ModelFollow struct {
	model    ModelSource
	wake     IndexWake
	recorder modelRecorder
	rebuild  rebuilder
}

// NewModelFollow makes the follow over the model of the embedder, the worker that indexes the vault, the
// runtime that keeps the model and the rebuild of the passes. wake may be nil.
func NewModelFollow(model ModelSource, wake IndexWake, runtime *Runtime, rebuild RebuildSession) ModelFollow {
	return ModelFollow{model: model, wake: wake, recorder: runtime, rebuild: rebuild}
}

func (f ModelFollow) start(ctx context.Context) {
	if f.model == nil {
		return
	}
	f.model.OnModelChange(func(isChunkSizeChanged bool) {
		f.recorder.RecordModel(f.model.Describe())
		if f.wake != nil {
			f.wake.WakeSources()
		}
		if isChunkSizeChanged {
			f.rebuild.Execute(ctx)
		}
	})
}

// StartWindow opens the first vault of a window and follows the model of its embedder.
type StartWindow struct {
	diag          *Diagnostics
	embedderError error
	follow        ModelFollow
	session       vaultSession
	isRebuilding  bool
}

// NewStartWindow makes the use case over the diagnostics, the session and the model follow of the window.
// embedderError is why the window has no embedder, or nil. isRebuilding is whether every file is indexed
// again for this launch.
func NewStartWindow(diag *Diagnostics, embedderError error, follow ModelFollow, session *Session, isRebuilding bool) StartWindow {
	return StartWindow{diag: diag, embedderError: embedderError, follow: follow, session: session, isRebuilding: isRebuilding}
}

// Execute opens first. Reindexing every file applies to this launch only; it is not carried to a
// vault opened later.
func (u StartWindow) Execute(ctx context.Context, first domain.Vault) {
	if why := u.embedderError; why != nil {
		u.diag.say("not embedding: %v\n", why)
		u.diag.tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: why.Error()})
	}
	u.follow.start(ctx)
	u.session.startVault(ctx, first, u.isRebuilding)
}
