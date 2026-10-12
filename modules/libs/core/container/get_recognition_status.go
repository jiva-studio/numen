package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// GetRecognitionStatus is nil where there is no worker to ask.
func (c Config) GetRecognitionStatus(worker *source.RecognitionWorker) *source.GetRecognitionStatus {
	if worker == nil {
		return nil
	}
	return source.NewGetRecognitionStatus(worker)
}
