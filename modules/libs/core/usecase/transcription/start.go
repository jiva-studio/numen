package transcription

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Start queues the transcription of one recording and returns
// without waiting for it to finish.
type Start struct {
	worker *Worker
}

func NewStart(worker *Worker) *Start {
	return &Start{worker: worker}
}

func (u *Start) Execute(v domain.Vault, path string) port.StartOutcome {
	return u.worker.Start(v, path)
}
