package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/transcription"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// TranscriberReady says whether a recording could be listened to now without
// waiting for anything to arrive. Which models those are is settled here, with
// every other choice of adapter.
func (c Config) TranscriberReady() bool { return transcription.Ready(c.Transcription) }

// Transcriber is what listens to a recording on this machine, opened now: the
// transcriber, what gives it back, and why there is none.
//
// It waits for whatever is missing, so it is for a terminal, where waiting is
// what a person came for. A window asks Transcribing instead.
func (c Config) Transcriber(ctx context.Context) (transcriber port.Transcriber, close func() error, why error) {
	models, err := transcription.Open(ctx, c.Transcription)
	if err != nil {
		return nil, nil, err
	}
	return models, models.Close, nil
}

// Transcribe is one recording listened to with the transcriber given, for a
// caller that asked for that recording and waits for it. Transcribing is the
// queue that listens to what nobody asked about.
func (c Config) Transcribe(sources port.SourceRepository, by port.Transcriber) source.Transcribe {
	return source.NewTranscribe(c.VaultReaders(), sources, c.DerivedStores(), by)
}

// Transcribing is the queue that listens to this installation's recordings,
// built against the adapters it was configured with and reporting itself into
// the list of what is being done.
//
// models is what it takes its turn at with the queue that reads scans: one run
// holds this machine's models at a time.
func (c Config) Transcribing(
	ctx context.Context,
	sources port.SourceRepository,
	tasks *task.Tasks,
	models *source.Lock,
) *source.TranscriptionWorker {
	return source.NewTranscriptionWorker(ctx, source.Transcriptions{
		Readers: c.VaultReaders(),
		Derived: c.DerivedStores(),
		Sources: sources,
		Tasks:   tasks,
		Models:  models,
		Runtime: source.TranscriptionRuntime{
			Open: func(
				ctx context.Context, tell func(what string, done, total int64),
			) (port.Transcriber, func() error, error) {
				cfg := c.Transcription
				cfg.Progress = tell
				models, err := transcription.Open(ctx, cfg)
				if err != nil {
					return nil, nil, err
				}
				return models, models.Close, nil
			},
			Ready: c.TranscriberReady,
		},
		Unasked:      c.TranscribesUnder,
		Proofreading: c.proofreadingFor(c.SpeechProofreading),
	})
}
