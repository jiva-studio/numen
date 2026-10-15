package main

import (
	"context"
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
)

type lifecycleServer struct {
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

type recordedAudio struct {
	duration int32
	segments []*numenv1.AudioSegment
}

type transcriptionServer struct {
	mu         sync.Mutex
	nextID     int64
	recordings map[string]*recordedAudio
}

func newTranscriptionServer() *transcriptionServer {
	return &transcriptionServer{
		recordings: make(map[string]*recordedAudio),
	}
}

func (s *transcriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[numenv1.GetTranscriptionModelRequest],
) (*connect.Response[numenv1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&numenv1.GetTranscriptionModelResponse{
		Model:     "parakeet-tdt-0.6b-v3-int8",
		Segmenter: "silero-vad",
		Cutting:   "0.50/500/200/30000/100/2500",
		From:      "builtin",
	}), nil
}

func (s *transcriptionServer) OpenRecording(
	_ context.Context,
	req *connect.Request[numenv1.OpenRecordingRequest],
) (*connect.Response[numenv1.OpenRecordingResponse], error) {
	data := req.Msg.GetAudioData()
	durationMs := int32(len(data) / 32)
	if durationMs < 1000 {
		durationMs = 1000
	}

	s.mu.Lock()
	s.nextID++
	id := fmt.Sprintf("rec-%d", s.nextID)
	s.recordings[id] = &recordedAudio{
		duration: durationMs,
		segments: []*numenv1.AudioSegment{
			{
				FromMs:  0,
				ToMs:    durationMs,
				Samples: make([]float32, 160),
			},
		},
	}
	s.mu.Unlock()

	return connect.NewResponse(&numenv1.OpenRecordingResponse{
		RecordingId: id,
		DurationMs:  durationMs,
	}), nil
}

func (s *transcriptionServer) GetSegments(
	_ context.Context,
	req *connect.Request[numenv1.GetSegmentsRequest],
) (*connect.Response[numenv1.GetSegmentsResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.recordings[req.Msg.GetRecordingId()]
	if !ok {
		return connect.NewResponse(&numenv1.GetSegmentsResponse{
			Segments: nil,
		}), nil
	}

	from := req.Msg.GetFromMs()
	count := int(req.Msg.GetCount())
	var out []*numenv1.AudioSegment
	for _, seg := range rec.segments {
		if seg.GetFromMs() < from {
			continue
		}
		out = append(out, seg)
		if count > 0 && len(out) == count {
			break
		}
	}

	return connect.NewResponse(&numenv1.GetSegmentsResponse{
		Segments: out,
	}), nil
}

func (s *transcriptionServer) Transcribe(
	_ context.Context,
	req *connect.Request[numenv1.TranscribeRequest],
) (*connect.Response[numenv1.TranscribeResponse], error) {
	if len(req.Msg.GetSamples()) == 0 {
		return connect.NewResponse(&numenv1.TranscribeResponse{
			Text: "",
		}), nil
	}

	return connect.NewResponse(&numenv1.TranscribeResponse{
		Text: "Transcribed speech from native transcription plugin.",
	}), nil
}

func (s *transcriptionServer) CloseRecording(
	_ context.Context,
	req *connect.Request[numenv1.CloseRecordingRequest],
) (*connect.Response[numenv1.CloseRecordingResponse], error) {
	s.mu.Lock()
	delete(s.recordings, req.Msg.GetRecordingId())
	s.mu.Unlock()

	return connect.NewResponse(&numenv1.CloseRecordingResponse{
		Success: true,
	}), nil
}

func main() {
	var portFlag int
	var tokenFlag string

	flag.IntVar(&portFlag, "port", 0, "Port to listen on")
	flag.StringVar(&tokenFlag, "token", "", "Authentication token")
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

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", portFlag))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to listen: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port

	shutdownCh := make(chan struct{})
	life := &lifecycleServer{shutdownTrigger: shutdownCh}
	trans := newTranscriptionServer()

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
	_ = srv.Shutdown(ctx)
}
