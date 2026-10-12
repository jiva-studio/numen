package container

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// mockTranscriber answers recordings with synthetic cues without running inference.
type mockTranscriber struct{}

// NewMockTranscriber creates a mock transcriber for test runs.
func NewMockTranscriber() port.Transcriber {
	return &mockTranscriber{}
}

func (m *mockTranscriber) GetTranscriptionModel() port.TranscriptionModel {
	return port.TranscriptionModel{
		Model:     "mock-asr",
		Segmenter: "mock-vad",
		From:      "test",
	}
}

func (m *mockTranscriber) Open(ctx context.Context, raw []byte) (port.Recording, error) {
	return &mockRecording{duration: 2000}, nil
}

func (m *mockTranscriber) Transcribe(ctx context.Context, audio port.Audio) (string, error) {
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

func (r *mockRecording) GetSegments(ctx context.Context, from, count int) ([]port.Audio, error) {
	if r.hasServed {
		return nil, nil
	}
	r.hasServed = true
	return []port.Audio{
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
