package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/model"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

func newOpener(runtime engine.Runtime, isMock bool) model.Opener {
	return func(ctx context.Context, cfg settings.Config) (model.Model, error) {
		if isMock {
			return engine.NewMockTranscriber(), nil
		}
		return engine.Open(ctx, runtime, cfg)
	}
}
