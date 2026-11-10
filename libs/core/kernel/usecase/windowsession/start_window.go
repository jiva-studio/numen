package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// StartWindow opens the first vault of a window and follows the model of its embedder.
type StartWindow struct {
	life    *lifecycle
	rebuild RebuildSession
}

// Execute opens first. Reindexing every file applies to this launch only; it is not carried to a
// vault opened later.
func (u StartWindow) Execute(ctx context.Context, first domain.Vault) {
	if why := u.life.deps.EmbedderError; why != nil {
		u.life.say("not embedding: %v\n", why)
		u.life.deps.Tasks.AddTask(work.Task{ID: work.MakingVectors, Title: "Indexing", Error: why.Error()})
	}
	u.followModel()
	u.life.startVault(ctx, first, u.life.deps.IsRebuilding)
}

// followModel records the model the embedder announces, and builds the passes again when its chunk
// size changes, because the notes are then cut again.
func (u StartWindow) followModel() {
	deps := u.life.deps
	if deps.Model == nil {
		return
	}
	deps.Model.OnModelChange(func(isChunkSizeChanged bool) {
		u.life.rt.RecordModel(deps.Model.Describe())
		if deps.Wake != nil {
			deps.Wake.WakeSources()
		}
		if isChunkSizeChanged {
			u.rebuild.Execute()
		}
	})
}
