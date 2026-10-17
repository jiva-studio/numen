package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/engine"
)

type engineFactory func(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error)

func defaultEngineFactory(ctx context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
	if os.Getenv("NUMEN_MOCK_ML") == "true" {
		return engine.NewMockTranscriber(), nil
	}
	return engine.Open(ctx, cfg)
}

type transcriptionServer struct {
	mu         sync.Mutex
	cfg        engine.Config
	factory    engineFactory
	engine     engine.TranscriberPort
	nextID     int64
	recordings map[string]engine.Recording
}

func (s *transcriptionServer) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lang, ok := values["language"]; ok && lang != "" {
		s.cfg.Model.Name = lang
	}
}

func newTranscriptionServer(cfg engine.Config, factory engineFactory) *transcriptionServer {
	return &transcriptionServer{
		cfg:        cfg,
		factory:    factory,
		recordings: make(map[string]engine.Recording),
	}
}

func (s *transcriptionServer) getEngine(ctx context.Context) (engine.TranscriberPort, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine != nil {
		return s.engine, nil
	}
	e, err := s.factory(ctx, s.cfg)
	if err != nil {
		return nil, err
	}
	s.engine = e
	return s.engine, nil
}

func (s *transcriptionServer) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.recordings {
		_ = rec.Close()
	}
	clear(s.recordings)
	if s.engine != nil {
		_ = s.engine.Close()
		s.engine = nil
	}
	return nil
}

func (s *transcriptionServer) GetTranscriptionModel(
	ctx context.Context,
	_ *connect.Request[numenv1.GetTranscriptionModelRequest],
) (*connect.Response[numenv1.GetTranscriptionModelResponse], error) {
	s.mu.Lock()
	e := s.engine
	s.mu.Unlock()

	if e != nil {
		m := e.GetTranscriptionModel()
		return connect.NewResponse(&numenv1.GetTranscriptionModelResponse{
			Model:     m.Model,
			Segmenter: m.Segmenter,
			Cutting:   m.Cutting,
			From:      m.From,
		}), nil
	}

	return connect.NewResponse(&numenv1.GetTranscriptionModelResponse{
		Model:     s.cfg.Model.Name,
		Segmenter: s.cfg.Segmenter.Name,
		Cutting:   s.cfg.Segmenter.DescribeSegmentSettings(),
		From:      "builtin",
	}), nil
}

func validateVaultPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty audio path")
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return "", errors.New("path traversal outside allowed scope")
	}

	vaultRoot := os.Getenv("NUMEN_VAULT_PATH")
	if vaultRoot != "" {
		cleanVault := filepath.Clean(vaultRoot)
		if !filepath.IsAbs(cleanPath) {
			cleanPath = filepath.Join(cleanVault, cleanPath)
		}
		prefix := cleanVault
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
		if cleanPath != cleanVault && !strings.HasPrefix(cleanPath, prefix) {
			return "", errors.New("path outside allowed vault root")
		}
	}
	return cleanPath, nil
}

func readAudioFile(path string) ([]byte, error) {
	cleanPath, err := validateVaultPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleanPath)
}

func (s *transcriptionServer) OpenRecording(
	ctx context.Context,
	req *connect.Request[numenv1.OpenRecordingRequest],
) (*connect.Response[numenv1.OpenRecordingResponse], error) {
	var data []byte
	if path := req.Msg.GetPath(); path != "" {
		var err error
		data, err = readAudioFile(path)
		if err != nil {
			if strings.Contains(err.Error(), "vault") || strings.Contains(err.Error(), "traversal") {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read audio %s: %w", path, err))
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("read audio %s: %w", path, err))
		}
	} else {
		data = req.Msg.GetAudioData()
	}

	if len(data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("empty audio data"))
	}

	t, err := s.getEngine(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("transcriber open: %w", err))
	}

	rec, err := t.Open(ctx, data)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("open recording: %w", err))
	}

	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("rec-%d", s.nextID)
	s.recordings[id] = rec
	s.mu.Unlock()

	return connect.NewResponse(&numenv1.OpenRecordingResponse{
		RecordingId: id,
		DurationMs:  int32(rec.GetDuration()),
	}), nil
}

func (s *transcriptionServer) GetSegments(
	ctx context.Context,
	req *connect.Request[numenv1.GetSegmentsRequest],
) (*connect.Response[numenv1.GetSegmentsResponse], error) {
	s.mu.Lock()
	rec, ok := s.recordings[req.Msg.GetRecordingId()]
	s.mu.Unlock()

	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("recording %q not found", req.Msg.GetRecordingId()))
	}

	found, err := rec.GetSegments(ctx, int(req.Msg.GetFromMs()), int(req.Msg.GetCount()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get segments: %w", err))
	}

	segments := make([]*numenv1.AudioSegment, 0, len(found))
	for _, one := range found {
		segments = append(segments, &numenv1.AudioSegment{
			FromMs:  int32(one.From),
			ToMs:    int32(one.To),
			Samples: one.Samples,
		})
	}

	return connect.NewResponse(&numenv1.GetSegmentsResponse{
		Segments: segments,
	}), nil
}

func (s *transcriptionServer) Transcribe(
	ctx context.Context,
	req *connect.Request[numenv1.TranscribeRequest],
) (*connect.Response[numenv1.TranscribeResponse], error) {
	samples := req.Msg.GetSamples()
	if len(samples) == 0 {
		return connect.NewResponse(&numenv1.TranscribeResponse{
			Text: "",
		}), nil
	}

	t, err := s.getEngine(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("transcriber open: %w", err))
	}

	text, err := t.Transcribe(ctx, engine.Audio{
		Samples: samples,
		From:    int(req.Msg.GetFromMs()),
		To:      int(req.Msg.GetToMs()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("transcribe: %w", err))
	}

	return connect.NewResponse(&numenv1.TranscribeResponse{
		Text: text,
	}), nil
}

func (s *transcriptionServer) CloseRecording(
	_ context.Context,
	req *connect.Request[numenv1.CloseRecordingRequest],
) (*connect.Response[numenv1.CloseRecordingResponse], error) {
	s.mu.Lock()
	rec, ok := s.recordings[req.Msg.GetRecordingId()]
	delete(s.recordings, req.Msg.GetRecordingId())
	s.mu.Unlock()

	if ok && rec != nil {
		_ = rec.Close()
	}

	return connect.NewResponse(&numenv1.CloseRecordingResponse{
		Success: true,
	}), nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var modelsDirFlag string
	var runtimeFlag string

	flag.StringVar(&modelsDirFlag, "models-dir", "", "Directory holding model weights")
	flag.StringVar(&runtimeFlag, "runtime", "", "Path to ONNX runtime library")

	s := server.New()

	cfg := engine.Defaults()
	if modelsDirFlag != "" {
		cfg.Dir = modelsDirFlag
	} else if envDir := os.Getenv(models.EnvDir); envDir != "" {
		cfg.Dir = envDir
	}
	if runtimeFlag != "" {
		cfg.Runtime = runtimeFlag
	} else if envRuntime := os.Getenv("NUMEN_ONNX_RUNTIME"); envRuntime != "" {
		cfg.Runtime = envRuntime
	}

	trans := newTranscriptionServer(cfg, defaultEngineFactory)
	defer trans.close()

	s.Lifecycle().SetUpdateConfig(trans.updateConfig)
	s.Register(numenv1connect.NewTranscriptionServiceHandler(trans, s.HandlerOptions()...))

	return s.Serve()
}
