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

// StartWindow opens the first vault of a window and follows the model of its embedder.
type StartWindow struct {
	diag          *Diagnostics
	embedderError error
	model         ModelSource
	wake          IndexWake
	recorder      modelRecorder
	session       vaultSession
	rebuild       rebuilder
	isRebuilding  bool
}

// NewStartWindow makes the use case over the diagnostics, the runtime and the session of the window.
// embedderError is why the window has no embedder, or nil. model and wake may be nil. isRebuilding is
// whether every file is indexed again for this launch.
func NewStartWindow(
	diag *Diagnostics, embedderError error, model ModelSource, wake IndexWake, runtime *Runtime,
	session *Session, rebuild RebuildSession, isRebuilding bool,
) StartWindow {
	return StartWindow{
		diag: diag, embedderError: embedderError, model: model, wake: wake, recorder: runtime,
		session: session, rebuild: rebuild, isRebuilding: isRebuilding,
	}
}

// Execute opens first. Reindexing every file applies to this launch only; it is not carried to a
// vault opened later.
func (u StartWindow) Execute(ctx context.Context, first domain.Vault) {
	if why := u.embedderError; why != nil {
		u.diag.say("not embedding: %v\n", why)
		u.diag.tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: why.Error()})
	}
	u.followModel(ctx)
	u.session.startVault(ctx, first, u.isRebuilding)
}

func (u StartWindow) followModel(ctx context.Context) {
	if u.model == nil {
		return
	}
	u.model.OnModelChange(func(isChunkSizeChanged bool) {
		u.recorder.RecordModel(u.model.Describe())
		if u.wake != nil {
			u.wake.WakeSources()
		}
		if isChunkSizeChanged {
			u.rebuild.Execute(ctx)
		}
	})
}
