package service

import (
	"context"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/models"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// recognizer lets the transcription use case hear through the engine.
type recognizer struct {
	port engine.TranscriberPort
}

func (r recognizer) Open(ctx context.Context, raw []byte) (transcribe.Recording, error) {
	opened, err := r.port.Open(ctx, raw)
	if err != nil {
		return nil, err
	}
	return recording{Recording: opened}, nil
}

func (r recognizer) Transcribe(ctx context.Context, audio transcribe.Audio) (string, error) {
	return r.port.Transcribe(ctx, engine.Audio(audio))
}

type recording struct {
	engine.Recording
}

func (r recording) GetSegments(ctx context.Context, from, count int) ([]transcribe.Audio, error) {
	cut, err := r.Recording.GetSegments(ctx, from, count)
	if err != nil {
		return nil, err
	}
	segments := make([]transcribe.Audio, len(cut))
	for i, one := range cut {
		segments[i] = transcribe.Audio(one)
	}
	return segments, nil
}

func (r recognizer) Close() error {
	return r.port.Close()
}

func newOpener(runtime engine.Runtime, isMock bool) models.Opener {
	return func(ctx context.Context, cfg settings.Config) (models.Model, error) {
		if isMock {
			return recognizer{port: engine.NewMockTranscriber()}, nil
		}
		port, err := engine.Open(ctx, runtime, cfg)
		if err != nil {
			return nil, err
		}
		return recognizer{port: port}, nil
	}
}
