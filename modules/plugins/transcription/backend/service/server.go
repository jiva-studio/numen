// Package service is the backend of the transcription plugin.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/models"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault"
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

	// vaultRoot is the vault the audio a request names by path is read from; empty holds it to no vault.
	vaultRoot string
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
	e, err := s.factory(context.WithoutCancel(ctx), s.cfg)
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

func (s *transcriptionServer) readAudioFile(path string) ([]byte, error) {
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot, path)
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
		data, err = s.readAudioFile(path)
		if err != nil {
			if errors.Is(err, vault.ErrOutsideVault) || errors.Is(err, vault.ErrPathTraversal) {
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

// NewConfig is the engine configuration the environment gives, with the directory of the models and the
// ONNX runtime library a command line names laid over it.
func NewConfig(modelsDir, runtime string) engine.Config {
	cfg := engine.Defaults()
	if modelsDir != "" {
		cfg.Dir = modelsDir
	} else if envDir := os.Getenv(models.EnvDir); envDir != "" {
		cfg.Dir = envDir
	}
	if runtime != "" {
		cfg.Runtime = runtime
	} else if envRuntime := os.Getenv("NUMEN_ONNX_RUNTIME"); envRuntime != "" {
		cfg.Runtime = envRuntime
	}
	return cfg
}

// Mount registers the transcription service on s. Audio a request names by path is read from
// vaultRoot. The recordings and the engine close with s.
func Mount(s *server.Server, cfg engine.Config, vaultRoot string) error {
	trans := newTranscriptionServer(cfg, defaultEngineFactory)
	trans.vaultRoot = vaultRoot
	s.SetUpdateConfig(trans.updateConfig)
	s.Register(numenv1connect.NewTranscriptionServiceHandler(trans, s.HandlerOptions()...))
	s.OnClose(func() { _ = trans.close() })
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return Mount(s, NewConfig("", ""), vars["NUMEN_VAULT_PATH"])
	})
}
