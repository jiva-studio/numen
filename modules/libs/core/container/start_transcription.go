package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// StartTranscription is nil where there is no worker to start.
func (c Config) StartTranscription(worker *source.TranscriptionWorker) *source.StartTranscription {
	if worker == nil {
		return nil
	}
	return source.NewStartTranscription(worker)
}
