package engine

import "context"

// mockTranscriber is an in-memory synthetic speech recognition engine for automated tests.
type mockTranscriber struct{}

// NewMockTranscriber creates a fast synthetic speech recognizer.
func NewMockTranscriber() TranscriberPort {
	return &mockTranscriber{}
}

func (m *mockTranscriber) Open(_ context.Context, _ []byte) (Recording, error) {
	return &mockRecording{duration: 2000, hasServed: false}, nil
}

func (m *mockTranscriber) Transcribe(_ context.Context, _ Audio) (string, error) {
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

func (r *mockRecording) GetSegments(_ context.Context, _, _ int) ([]Audio, error) {
	if r.hasServed {
		return nil, nil
	}
	r.hasServed = true
	return []Audio{
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
