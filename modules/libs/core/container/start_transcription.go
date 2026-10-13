package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"

// StartTranscription is nil where there is no worker to start.
func (c Config) StartTranscription(worker *transcription.Worker) *transcription.Start {
	if worker == nil {
		return nil
	}
	return transcription.NewStart(worker)
}
