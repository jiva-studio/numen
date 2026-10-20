package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// restartedPlugin answers every service a bridge reaches, and counts the calls
// it was asked.
type restartedPlugin struct {
	numenv1connect.UnimplementedEmbeddingServiceHandler
	asked atomic.Int32
}

func (p *restartedPlugin) GetEmbeddingModel(
	context.Context, *connect.Request[v1.GetEmbeddingModelRequest],
) (*connect.Response[v1.GetEmbeddingModelResponse], error) {
	return connect.NewResponse(&v1.GetEmbeddingModelResponse{}), nil
}

func (p *restartedPlugin) EmbedBatch(
	context.Context, *connect.Request[v1.EmbedBatchRequest],
) (*connect.Response[v1.EmbedBatchResponse], error) {
	p.asked.Add(1)
	return connect.NewResponse(&v1.EmbedBatchResponse{}), nil
}

type restartedRecognition struct {
	numenv1connect.UnimplementedRecognitionServiceHandler
	plugin *restartedPlugin
}

func (r restartedRecognition) GetRecognitionModel(
	context.Context, *connect.Request[v1.GetRecognitionModelRequest],
) (*connect.Response[v1.GetRecognitionModelResponse], error) {
	return connect.NewResponse(&v1.GetRecognitionModelResponse{}), nil
}

func (r restartedRecognition) RecognisePage(
	context.Context, *connect.Request[v1.RecognisePageRequest],
) (*connect.Response[v1.RecognisePageResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.RecognisePageResponse{}), nil
}

type restartedTranscription struct {
	numenv1connect.UnimplementedTranscriptionServiceHandler
	plugin *restartedPlugin
}

func (r restartedTranscription) GetTranscriptionModel(
	context.Context, *connect.Request[v1.GetTranscriptionModelRequest],
) (*connect.Response[v1.GetTranscriptionModelResponse], error) {
	return connect.NewResponse(&v1.GetTranscriptionModelResponse{}), nil
}

func (r restartedTranscription) Transcribe(
	context.Context, *connect.Request[v1.TranscribeRequest],
) (*connect.Response[v1.TranscribeResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.TranscribeResponse{}), nil
}

type restartedDocument struct {
	numenv1connect.UnimplementedDocumentServiceHandler
	plugin *restartedPlugin
}

func (r restartedDocument) ExtractText(
	context.Context, *connect.Request[v1.ExtractTextRequest],
) (*connect.Response[v1.ExtractTextResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.ExtractTextResponse{}), nil
}

type restartedBook struct {
	numenv1connect.UnimplementedBookServiceHandler
	plugin *restartedPlugin
}

func (r restartedBook) GetBook(
	context.Context, *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.GetBookResponse{}), nil
}

func (r restartedBook) ReadBookEntry(
	context.Context, *connect.Request[v1.ReadBookEntryRequest],
) (*connect.Response[v1.ReadBookEntryResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.ReadBookEntryResponse{}), nil
}

func (r restartedBook) ExtractBookText(
	context.Context, *connect.Request[v1.ExtractBookTextRequest],
) (*connect.Response[v1.ExtractBookTextResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.ExtractBookTextResponse{}), nil
}

type restartedDownload struct {
	numenv1connect.UnimplementedDownloadServiceHandler
	plugin *restartedPlugin
}

func (r restartedDownload) GetMetadata(
	context.Context, *connect.Request[v1.GetMetadataRequest],
) (*connect.Response[v1.GetMetadataResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.GetMetadataResponse{}), nil
}

type restartedAgent struct {
	numenv1connect.UnimplementedAgentServiceHandler
	plugin *restartedPlugin
}

func (r restartedAgent) GetAgentPrograms(
	context.Context, *connect.Request[v1.GetAgentProgramsRequest],
) (*connect.Response[v1.GetAgentProgramsResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.GetAgentProgramsResponse{}), nil
}

// newRestartedPlugin serves every service on a port of its own.
func newRestartedPlugin(t *testing.T) (*restartedPlugin, *httptest.Server) {
	t.Helper()
	plugin := &restartedPlugin{}
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewEmbeddingServiceHandler(plugin))
	mux.Handle(numenv1connect.NewRecognitionServiceHandler(restartedRecognition{plugin: plugin}))
	mux.Handle(numenv1connect.NewTranscriptionServiceHandler(restartedTranscription{plugin: plugin}))
	mux.Handle(numenv1connect.NewDocumentServiceHandler(restartedDocument{plugin: plugin}))
	mux.Handle(numenv1connect.NewBookServiceHandler(restartedBook{plugin: plugin}))
	mux.Handle(numenv1connect.NewDownloadServiceHandler(restartedDownload{plugin: plugin}))
	mux.Handle(numenv1connect.NewAgentServiceHandler(restartedAgent{plugin: plugin}))
	mux.Handle(numenv1connect.NewProofreadingServiceHandler(restartedProofreading{plugin: plugin}))
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(restartedTools{plugin: plugin}))
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(testsupport.PluginLifecycle{}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return plugin, server
}

// restartingLauncher starts a process for each activation and a plugin for it
// on a port of its own, and keeps both in the order they were started.
type restartingLauncher struct {
	t       *testing.T
	mu      sync.Mutex
	procs   []*os.Process
	plugins []*restartedPlugin
	servers []*httptest.Server
}

func (l *restartingLauncher) launch(_ context.Context, _ DiscoveredPlugin, _ int, _ string) (*os.Process, io.ReadCloser, error) {
	cmd := newIdleCommand()
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	l.t.Cleanup(func() { killProcess(l.t, cmd.Process) })
	plugin, server := newRestartedPlugin(l.t)
	l.mu.Lock()
	l.procs = append(l.procs, cmd.Process)
	l.plugins = append(l.plugins, plugin)
	l.servers = append(l.servers, server)
	l.mu.Unlock()
	ready := fmt.Sprintf("NUMEN_READY port=%d\n", server.Listener.Addr().(*net.TCPAddr).Port)
	return cmd.Process, io.NopCloser(strings.NewReader(ready)), nil
}

func (l *restartingLauncher) get(at int) (*os.Process, *restartedPlugin, *httptest.Server) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.procs[at], l.plugins[at], l.servers[at]
}

func (l *restartingLauncher) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.procs)
}

// writeRestartedPlugin is a plugin declaring every capability a bridge looks up.
func writeRestartedPlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	at := filepath.Join(dir, "stub")
	if err := os.MkdirAll(at, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"id": "md.numen.stub",
		"name": "All",
		"version": "1.0.0",
		"capabilities": ["embedding", "recognition", "transcription", "document", "book", "downloader", "proofreading", "tools"],
		"targets": ["desktop.editor"],
		"permissions": ["vault:read", "network:outbound"],
		"backend": {"type": "process", "entrypoint": "bin/stub"}
	}`
	if err := os.WriteFile(filepath.Join(at, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// A bridge opened before its plugin stopped reaches the plugin at the port it
// was started again on, at the next call.
func TestABridgeReachesItsPluginStartedAgainOnAnotherPort(t *testing.T) {
	for _, tc := range bridgeContracts {
		t.Run(tc.name, func(t *testing.T) {
			host := NewHost("desktop.editor", writeRestartedPlugin(t))
			if err := host.Discover(); err != nil {
				t.Fatal(err)
			}
			launcher := &restartingLauncher{t: t}
			host.SetLauncher(launcher.launch)

			bridge, call, err := tc.open(t.Context(), host)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer func() { _ = bridge.Close() }()
			if _, err := call(t.Context()); err != nil {
				t.Fatalf("the first call failed: %v", err)
			}

			first, _, firstServer := launcher.get(0)
			killProcess(t, first)
			waitUntil(t, "the stopped plugin to lose its endpoint", func() bool {
				host.mu.RLock()
				defer host.mu.RUnlock()
				_, isRegistered := host.endpoints[contractPluginID]
				return !isRegistered
			})
			firstServer.Close()

			if _, err := call(t.Context()); err != nil {
				t.Fatalf("the call after the restart failed: %v", err)
			}
			if got := launcher.count(); got != 2 {
				t.Fatalf("the plugin was started %d times, want 2", got)
			}
			if _, again, _ := launcher.get(1); again.asked.Load() != 1 {
				t.Errorf("the plugin started again was asked %d times, want 1", again.asked.Load())
			}
			if err := host.TeardownAll(t.Context(), 50*time.Millisecond); err != nil {
				t.Errorf("TeardownAll: %v", err)
			}
		})
	}
}

type restartedProofreading struct {
	numenv1connect.UnimplementedProofreadingServiceHandler
	plugin *restartedPlugin
}

func (r restartedProofreading) ProofreadText(
	context.Context, *connect.Request[v1.ProofreadTextRequest],
) (*connect.Response[v1.ProofreadTextResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.ProofreadTextResponse{Corrections: []*v1.ProofreadCorrection{{}}}), nil
}

type restartedTools struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	plugin *restartedPlugin
}

func (r restartedTools) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	r.plugin.asked.Add(1)
	return connect.NewResponse(&v1.ListToolsResponse{Tools: []*v1.PluginTool{{
		Name: "stub_read", InputSchema: []byte("{}"), Effect: v1.ToolEffect_TOOL_EFFECT_READS,
	}}}), nil
}
