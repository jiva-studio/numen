package service

import (
	"context"
	"os"
	"sync"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// envONNXRuntime names the variable that holds the path of the ONNX runtime library.
const envONNXRuntime = "NUMEN_ONNX_RUNTIME"

type openModel func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error)

func newParakeetOpener(runtime engine.Runtime) openModel {
	return func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
		return engine.Open(ctx, runtime, cfg)
	}
}

// models opens the model once, under the configuration the settings last named, and keeps it until
// the plugin stops.
type models struct {
	mu      sync.Mutex
	cfg     engine.Config
	isMock  bool
	open    openModel
	current engine.TranscriberPort
}

func (m *models) get(ctx context.Context) (engine.TranscriberPort, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil {
		return m.current, nil
	}
	if m.isMock {
		m.current = engine.NewMockTranscriber()
		return m.current, nil
	}
	loaded, err := m.open(context.WithoutCancel(ctx), m.cfg)
	if err != nil {
		return nil, err
	}
	m.current = loaded
	return m.current, nil
}

func (m *models) getRecognizer(ctx context.Context) (transcribe.Recognizer, error) {
	port, err := m.get(ctx)
	if err != nil {
		return nil, err
	}
	return recognizer{port: port}, nil
}

func (m *models) updateConfig(values map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if model, ok := engine.FindModel(values["model"]); ok {
		m.cfg.Model = model
	}
}

func (m *models) close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	err := m.current.Close()
	m.current = nil
	return err
}

func getConfig() engine.Config {
	cfg := engine.Defaults()
	if dir := os.Getenv(engine.EnvDir); dir != "" {
		cfg.Dir = dir
	}
	if runtime := os.Getenv(envONNXRuntime); runtime != "" {
		cfg.Runtime = runtime
	}
	return cfg
}

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
