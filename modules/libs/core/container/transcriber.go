package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
)

var (
	errTranscriptionUnavailable = errors.New("transcription plugin is unavailable or not installed")
)

// TranscriberReady says whether a recording could be listened to now without
// waiting for anything to arrive.
func (c Config) TranscriberReady() bool {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("transcription"); ok {
			return true
		}
	}
	return false
}

// Transcriber is what listens to a recording on this machine, opened now.
func (c Config) Transcriber(ctx context.Context) (transcriber port.Transcriber, closer func() error, why error) {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("transcription"); ok {
			bridge, err := plugin.OpenTranscriberBridge(ctx, c.PluginHost)
			if err != nil {
				return nil, nil, err
			}
			return bridge, bridge.Close, nil
		}
	}
	return nil, nil, errTranscriptionUnavailable
}

// Transcribe is one recording listened to with the transcriber given.
func (c Config) Transcribe(sources port.SourceRepository, by port.Transcriber) transcription.Transcribe {
	return transcription.NewTranscribe(c.VaultReaders(), sources, c.GetDerivedStores(), by)
}

// OpenTranscriptionWorker is the queue that listens to this installation's recordings.
func (c Config) OpenTranscriptionWorker(
	ctx context.Context,
	sources port.SourceRepository,
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
			Open:    c.openTranscriptionRuntime,
			Ready:   c.TranscriberReady,
			CanKeep: func() bool { return false },
		},
		Unasked:      c.TranscribesUnder,
		Proofreading: c.getProofreading(c.TranscriptProofreading),
		Wait:         c.GetWait(),
		Delay:        c.GetDelay(),
	})
}

func (c Config) openTranscriptionRuntime(
	ctx context.Context, _ func(what string, done, total int64),
) (port.Transcriber, func() error, error) {
	if c.PluginHost != nil && !c.SafeMode {
		if _, ok := c.PluginHost.FindPluginByCapability("transcription"); ok {
			bridge, err := plugin.OpenTranscriberBridge(ctx, c.PluginHost)
			if err != nil {
				return nil, nil, err
			}
			return bridge, bridge.Close, nil
		}
	}
	return nil, nil, errTranscriptionUnavailable
}
