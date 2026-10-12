package source

import (
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// StartRecognition queues the recognition of one scanned document and returns
// without waiting for it to finish.
type StartRecognition struct {
	worker *RecognitionWorker
}

func NewStartRecognition(worker *RecognitionWorker) *StartRecognition {
	return &StartRecognition{worker: worker}
}

func (u *StartRecognition) Execute(v domain.Vault, path string) port.StartOutcome {
	return u.worker.Start(v, path)
}
