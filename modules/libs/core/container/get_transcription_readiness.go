package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"

// GetTranscriptionReadiness is nil where there is no worker to ask.
func (c Config) GetTranscriptionReadiness(worker *transcription.Worker) *transcription.GetReadiness {
	if worker == nil {
		return nil
	}
	return transcription.NewGetReadiness(worker)
}
