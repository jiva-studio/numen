package container

import "github.com/jiva-studio/numen/modules/libs/core/usecase/source"

// StartRecognition is nil where there is no worker to start.
func (c Config) StartRecognition(worker *source.RecognitionWorker) *source.StartRecognition {
	if worker == nil {
		return nil
	}
	return source.NewStartRecognition(worker)
}
