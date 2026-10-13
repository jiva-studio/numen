package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"

// GetRecognitionStatus is nil where there is no worker to ask.
func (c Config) GetRecognitionStatus(worker *recognition.Worker) *recognition.GetStatus {
	if worker == nil {
		return nil
	}
	return recognition.NewGetStatus(worker)
}
