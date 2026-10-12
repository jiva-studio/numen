package source

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// StartTranscription queues the transcription of one recording and returns
// without waiting for it to finish.
type StartTranscription struct {
	worker *TranscriptionWorker
}

func NewStartTranscription(worker *TranscriptionWorker) *StartTranscription {
	return &StartTranscription{worker: worker}
}

func (u *StartTranscription) Execute(v domain.Vault, path string) port.StartOutcome {
	return u.worker.Start(v, path)
}
