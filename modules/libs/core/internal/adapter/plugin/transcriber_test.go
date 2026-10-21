package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type mockTranscriptionServer struct{}

func (s *mockTranscriptionServer) GetTranscriptionModel(
	_ context.Context,
	_ *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{
		Model:     "test-parakeet",
		Segmenter: "test-vad",
		Cutting:   "0.5/500/200/30000/100/2500",
		From:      "test-cache",
	}), nil
}

func (s *mockTranscriptionServer) OpenRecording(
	_ context.Context,
	req *connect.Request[v1.OpenRecordingRequest],
) (*connect.Response[v1.OpenRecordingResponse], error) {
	if len(req.Msg.GetAudioData()) == 0 && req.Msg.GetPath() == "" {
		return nil, fmt.Errorf("empty audio data and path")
	}
	return connect.NewResponse(&v1.OpenRecordingResponse{
		RecordingId: "rec-123",
		DurationMs:  60000,
	}), nil
}

func (s *mockTranscriptionServer) GetSegments(
	_ context.Context,
	req *connect.Request[v1.GetSegmentsRequest],
) (*connect.Response[v1.GetSegmentsResponse], error) {
	if req.Msg.GetRecordingId() != "rec-123" {
		return nil, fmt.Errorf("unknown recording id")
	}
	return connect.NewResponse(&v1.GetSegmentsResponse{
		Segments: []*v1.AudioSegment{
			{
				FromMs:  100,
				ToMs:    2500,
				Samples: []float32{0.1, 0.2, 0.3},
			},
		},
	}), nil
}

func (s *mockTranscriptionServer) Transcribe(
	_ context.Context,
	req *connect.Request[v1.TranscribeRequest],
) (*connect.Response[v1.TranscribeResponse], error) {
	if len(req.Msg.GetSamples()) == 0 {
		return connect.NewResponse(&v1.TranscribeResponse{Text: ""}), nil
	}
	return connect.NewResponse(&v1.TranscribeResponse{
		Text: "Hello world transcription",
	}), nil
}

func (s *mockTranscriptionServer) CloseRecording(
	_ context.Context,
	_ *connect.Request[v1.CloseRecordingRequest],
) (*connect.Response[v1.CloseRecordingResponse], error) {
	return connect.NewResponse(&v1.CloseRecordingResponse{
		Success: true,
	}), nil
}

func createTestTranscriptionPluginTree(t *testing.T, dir string) {
	t.Helper()
	pDir := filepath.Join(dir, "transcription")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := `{
		"id": "md.numen.engines.transcription",
		"name": "Transcription Plugin",
		"version": "1.0.0",
		"description": "Transcription",
		"capabilities": ["transcription"],
		"targets": ["desktop.editor"],
		"activationEvents": ["onFile:*.mp3", "onFile:*.wav", "onCommand:transcribe"],
		"permissions": ["vault:read"],
		"lifecycle": {
			"mode": "on-demand",
			"idleTimeoutSeconds": 1,
			"keepAliveWhileVisible": false
		},
		"backend": {
			"type": "process",
			"entrypoint": "bin/transcription",
			"serviceName": "numen.v1.TranscriptionService"
		},
		"frontend": {
			"entrypoint": "dist/index.js",
			"styles": "dist/styles.css"
		},
		"settings": []
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func TestTranscriberBridge(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(&mockTranscriptionServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestTranscriptionPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := OpenTranscriberBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenTranscriberBridge failed: %v", err)
	}
	defer bridge.Close()

	model := bridge.GetTranscriptionModel()
	if model.Model != "test-parakeet" || model.Segmenter != "test-vad" {
		t.Fatalf("unexpected model: %+v", model)
	}

	rec, err := bridge.Open(ctx, []byte("fake-audio-bytes"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer rec.Close()

	if rec.GetDuration() != 60000 {
		t.Fatalf("expected duration 60000, got %d", rec.GetDuration())
	}

	segments, err := rec.GetSegments(ctx, 0, 10)
	if err != nil {
		t.Fatalf("GetSegments failed: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}
	if segments[0].From != 100 || segments[0].To != 2500 {
		t.Fatalf("unexpected segment timings: %+v", segments[0])
	}

	text, err := bridge.Transcribe(ctx, port.Audio{
		Samples: segments[0].Samples,
		From:    segments[0].From,
		To:      segments[0].To,
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	if text != "Hello world transcription" {
		t.Fatalf("unexpected transcribed text: %s", text)
	}
}

func TestTranscriberBridgeNoPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	_, err := OpenTranscriberBridge(t.Context(), host)
	if err == nil {
		t.Fatal("expected error when no plugin registered, got nil")
	}
}

type testNamedSource struct {
	name string
}

func (s testNamedSource) Name() string {
	return s.name
}

func TestTranscriberBridgeOpenPathAndNamed(t *testing.T) {
	mux := http.NewServeMux()
	testsupport.HandlePluginLifecycle(mux)
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(&mockTranscriptionServer{}))

	server := httptest.NewServer(mux)
	defer server.Close()

	serverPort := server.Listener.Addr().(*net.TCPAddr).Port

	tmp := t.TempDir()
	createTestTranscriptionPluginTree(t, tmp)

	host := NewHost("desktop.editor", tmp)
	host.SetVaultPath(tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mock proc failed: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	host.SetLauncher(func(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		r := strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", serverPort))
		return cmd.Process, io.NopCloser(r), nil
	})

	ctx := t.Context()
	bridge, err := OpenTranscriberBridge(ctx, host)
	if err != nil {
		t.Fatalf("OpenTranscriberBridge failed: %v", err)
	}
	defer bridge.Close()

	// 1. Open via path
	rec1, err := bridge.OpenPath(ctx, "audio/test.mp3")
	if err != nil {
		t.Fatalf("OpenPath failed: %v", err)
	}
	defer rec1.Close()
	if rec1.GetDuration() != 60000 {
		t.Fatalf("expected 60000 duration, got %d", rec1.GetDuration())
	}

	// 2. Open via named source
	rec2, err := bridge.OpenSource(ctx, testNamedSource{name: "audio/podcast.wav"})
	if err != nil {
		t.Fatalf("OpenSource with named source failed: %v", err)
	}
	defer rec2.Close()
	if rec2.GetDuration() != 60000 {
		t.Fatalf("expected 60000 duration, got %d", rec2.GetDuration())
	}

	// 3. Path traversal outside vault should be rejected
	_, err = bridge.OpenPath(ctx, "../outside.mp3")
	if err == nil {
		t.Fatal("expected error on path traversal outside vault, got nil")
	}
}

func TestPluginRecordingConcurrentClose(t *testing.T) {
	host := NewHost(t.TempDir(), "desktop.editor")
	rec := &pluginRecording{
		client:      nil,
		recordingID: "rec-test",
		durationMs:  1000,
		host:        host,
		pluginID:    "test-plugin",
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rec.Close()
		}()
	}
	wg.Wait()

	if rec.recordingID != "" {
		t.Errorf("expected empty recordingID after Close, got %s", rec.recordingID)
	}
}
