package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// GetTranscriptionReadiness is nil where there is no worker to ask.
func (c Config) GetTranscriptionReadiness(worker *source.TranscriptionWorker) *source.GetTranscriptionReadiness {
	if worker == nil {
		return nil
	}
	return source.NewGetTranscriptionReadiness(worker)
}
