package service

import (
	"context"

	modelprocess "github.com/jiva-studio/numen/libs/plugin/sdk-go/process"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/model"
)

func newOpener(runtime engine.Runtime, isMock bool, fetch engine.FetchModelFile, sessions *modelprocess.ModelSessions) model.Opener {
	return func(ctx context.Context, cfg settings.Config) (model.Model, error) {
		if isMock {
			return engine.NewMockTranscriber(), nil
		}
		compute := sessions.GetCompute()
		cfg.Provider, cfg.ThreadLimit = compute.Provider, compute.ThreadLimit
		recognized, err := engine.Open(ctx, runtime, cfg, fetch)
		if err != nil {
			return nil, err
		}
		return activeTranscriber{Model: recognized, release: sessions.Track(ctx, recognized.Providers)}, nil
	}
}

type activeTranscriber struct {
	model.Model
	release func()
}

func (m activeTranscriber) Close() error {
	defer m.release()
	return m.Model.Close()
}
