package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/engine"
)

type lifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
	shutdownTrigger chan struct{}
}

func (s *lifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[numenv1.PingRequest],
) (*connect.Response[numenv1.PingResponse], error) {
	return connect.NewResponse(&numenv1.PingResponse{
		Status: "ok",
	}), nil
}

func (s *lifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[numenv1.ShutdownRequest],
) (*connect.Response[numenv1.ShutdownResponse], error) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(s.shutdownTrigger)
	}()
	return connect.NewResponse(&numenv1.ShutdownResponse{
		Success: true,
	}), nil
}

type transcriptionServer struct {
	mu          sync.Mutex
	cfg         engine.Config
	transcriber *engine.Transcriber
	nextID      int64
	recordings  map[string]engine.Recording
}

func newTranscriptionServer(cfg engine.Config) *transcriptionServer {
	return &transcriptionServer{
		cfg:        cfg,
		recordings: make(map[string]engine.Recording),
	}
}

func (s *transcriptionServer) getTranscriber(ctx context.Context) (*engine.Transcriber, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.transcriber != nil {
		return s.transcriber, nil
	}
	t, err := engine.Open(ctx, s.cfg)
	if err != nil {
		return nil, err
	}
	s.transcriber = t
	return s.transcriber, nil
}

func (s *transcriptionServer) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.recordings {
		_ = rec.Close()
	}
	clear(s.recordings)
	if s.transcriber != nil {
		_ = s.transcriber.Close()
		s.transcriber = nil
	}
	return nil
}

func (s *transcriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetTranscriptionModelRequest],
) (*connect.Response[numenv1.GetTranscriptionModelResponse], error) {
	s.mu.Lock()
	t := s.transcriber
	s.mu.Unlock()

	if t != nil {
		m := t.GetTranscriptionModel()
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

func (s *transcriptionServer) OpenRecording(
	ctx context.Context,
	req *connect.Request[numenv1.OpenRecordingRequest],
) (*connect.Response[numenv1.OpenRecordingResponse], error) {
	data := req.Msg.GetAudioData()
	if len(data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("empty audio data"))
	}

	t, err := s.getTranscriber(ctx)
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

	t, err := s.getTranscriber(ctx)
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
	var portFlag int
	var tokenFlag string
	var modelsDirFlag string
	var runtimeFlag string

	flag.IntVar(&portFlag, "port", 0, "Port to listen on")
	flag.StringVar(&tokenFlag, "token", "", "Authentication token")
	flag.StringVar(&modelsDirFlag, "models-dir", "", "Directory holding model weights")
	flag.StringVar(&runtimeFlag, "runtime", "", "Path to ONNX runtime library")
	flag.Parse()

	if portFlag == 0 {
		if envPort := os.Getenv("NUMEN_PLUGIN_PORT"); envPort != "" {
			if p, err := strconv.Atoi(envPort); err == nil {
				portFlag = p
			}
		}
	}
	if tokenFlag == "" {
		tokenFlag = os.Getenv("NUMEN_PLUGIN_TOKEN")
	}

	cfg := engine.Defaults()
	if modelsDirFlag != "" {
		cfg.Dir = modelsDirFlag
	} else if envDir := os.Getenv("NUMEN_MODELS_DIR"); envDir != "" {
		cfg.Dir = envDir
	}
	if runtimeFlag != "" {
		cfg.Runtime = runtimeFlag
	} else if envRuntime := os.Getenv("NUMEN_ONNX_RUNTIME"); envRuntime != "" {
		cfg.Runtime = envRuntime
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	trans := newTranscriptionServer(cfg)
	defer trans.close()

	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(trans))

	handler := h2c.NewHandler(mux, &http2.Server{})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		}
	}()

	// Output ready handshake for plugin host
	fmt.Printf("NUMEN_READY port=%d\n", actualPort)
	os.Stdout.Sync()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-shutdownCh:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = trans.close()
	_ = srv.Shutdown(ctx)
}
