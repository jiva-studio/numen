package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/transcriptionport"
)

// TranscriberReady says whether a recording could be listened to now without
// waiting for anything to arrive.
func (c Config) TranscriberReady() bool {
	return c.IsMockML || c.hasPlugin(plugin.CapabilityTranscription)
}

// Transcriber is what listens to a recording on this machine, opened now.
func (c Config) Transcriber(ctx context.Context) (transcriber transcriptionport.Transcriber, closer func() error, why error) {
	if c.IsMockML {
		mock := NewMockTranscriber()
		return mock, mock.Close, nil
	}
	host, err := c.resolvePlugin(plugin.CapabilityTranscription)
	if err != nil {
		return nil, nil, err
	}
	bridge, err := plugin.OpenTranscriberBridge(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	return bridge, bridge.Close, nil
}

// Transcribe is one recording listened to with the transcriber given.
func (c Config) Transcribe(sources sourceport.SourceRepository, transcriber transcriptionport.Transcriber) transcription.Transcribe {
	return transcription.NewTranscribe(c.VaultReaders(), sources, c.GetDerivedStores(), transcriber)
}

// OpenTranscriptionWorker is the queue that listens to this installation's recordings.
func (c Config) OpenTranscriptionWorker(
	ctx context.Context,
	sources sourceport.SourceRepository,
	tasks *run.Tasks,
	models *run.Lock,
) *transcription.Worker {
	return transcription.NewWorker(ctx, transcription.Config{
		Readers: c.VaultReaders(),
		Derived: c.GetDerivedStores(),
		Sources: sources,
		Tasks:   tasks,
		Models:  models,
		Runtime: transcription.Runtime{
			Open: func(ctx context.Context, _ func(what string, finished, total int64)) (transcriptionport.Transcriber, func() error, error) {
				return c.Transcriber(ctx)
			},
			Ready:   c.TranscriberReady,
			CanKeep: func() bool { return false },
		},
		Unasked:      c.TranscribesUnder,
		Proofreading: c.getProofreading(c.TranscriptProofreading),
		Wait:         c.GetWait(),
		Delay:        c.GetDelay(),
	})
}
