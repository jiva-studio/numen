package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/transcriberbridge"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/transcriptionport"
)

// TranscriberReady says whether a recording could be listened to now without
// waiting for anything to arrive.
func (c Config) TranscriberReady() bool {
	return c.IsMockML || c.hasPlugin(bridge.CapabilityTranscription)
}

// Transcriber is what listens to a recording on this machine, opened now.
func (c Config) Transcriber(ctx context.Context) (transcriber transcriptionport.Transcriber, closer func() error, why error) {
	if c.IsMockML {
		mock := NewMockTranscriber()
		return mock, mock.Close, nil
	}
	host, err := c.resolvePlugin(bridge.CapabilityTranscription)
	if err != nil {
		return nil, nil, err
	}
	bridge, err := transcriberbridge.Open(ctx, host)
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

// mockTranscriber answers recordings with synthetic cues without running inference.
type mockTranscriber struct{}

// NewMockTranscriber creates a mock transcriber for test runs.
func NewMockTranscriber() transcriptionport.Transcriber {
	return &mockTranscriber{}
}

func (m *mockTranscriber) GetTranscriptionModel() transcriptionport.TranscriptionModel {
	return transcriptionport.TranscriptionModel{
		Model:     "mock-asr",
		Segmenter: "mock-vad",
		Cutting:   "mock-threshold",
		From:      "test",
	}
}

func (m *mockTranscriber) Open(_ context.Context, _ []byte) (transcriptionport.Recording, error) {
	return &mockRecording{duration: 2000, hasServed: false}, nil
}

func (m *mockTranscriber) Transcribe(_ context.Context, _ transcriptionport.Audio) (string, error) {
	return "Mock transcribed text for automated test.", nil
}

func (m *mockTranscriber) Close() error {
	return nil
}

type mockRecording struct {
	duration  int
	hasServed bool
}

func (r *mockRecording) GetDuration() int {
	return r.duration
}

func (r *mockRecording) GetSegments(_ context.Context, _, _ int) ([]transcriptionport.Audio, error) {
	if r.hasServed {
		return nil, nil
	}
	r.hasServed = true
	return []transcriptionport.Audio{
		{
			Samples: make([]float32, 16000),
			From:    0,
			To:      1000,
		},
	}, nil
}

func (r *mockRecording) Close() error {
	return nil
}
