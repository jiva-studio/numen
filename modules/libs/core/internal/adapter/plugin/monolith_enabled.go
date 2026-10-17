//go:build plugins_monolith

package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var (
	monolithMu           sync.RWMutex
	monolithRegistry     = make(map[string]http.Handler)
	documentHandler      numenv1connect.DocumentServiceHandler
	recognitionHandler   numenv1connect.RecognitionServiceHandler
	bookHandler          numenv1connect.BookServiceHandler
	transcriptionHandler numenv1connect.TranscriptionServiceHandler
	embeddingHandler     numenv1connect.EmbeddingServiceHandler
	downloadHandler      numenv1connect.DownloadServiceHandler
	agentHandler         numenv1connect.AgentServiceHandler
	proofreadingHandler  numenv1connect.ProofreadingServiceHandler
)

// RegisterDocumentServiceHandler sets the in-process DocumentServiceHandler.
func RegisterDocumentServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		documentHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.DocumentServiceHandler); ok {
		documentHandler = handler
	}
}

// RegisterRecognitionServiceHandler sets the in-process RecognitionServiceHandler.
func RegisterRecognitionServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		recognitionHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.RecognitionServiceHandler); ok {
		recognitionHandler = handler
	}
}

// RegisterBookServiceHandler sets the in-process BookServiceHandler.
func RegisterBookServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		bookHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.BookServiceHandler); ok {
		bookHandler = handler
	}
}

// RegisterTranscriptionServiceHandler sets the in-process TranscriptionServiceHandler.
func RegisterTranscriptionServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		transcriptionHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.TranscriptionServiceHandler); ok {
		transcriptionHandler = handler
	}
}

// RegisterEmbeddingServiceHandler sets the in-process EmbeddingServiceHandler.
func RegisterEmbeddingServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		embeddingHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.EmbeddingServiceHandler); ok {
		embeddingHandler = handler
	}
}

// RegisterDownloadServiceHandler sets the in-process DownloadServiceHandler.
func RegisterDownloadServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		downloadHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.DownloadServiceHandler); ok {
		downloadHandler = handler
	}
}

// RegisterAgentServiceHandler sets the in-process AgentServiceHandler.
func RegisterAgentServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		agentHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.AgentServiceHandler); ok {
		agentHandler = handler
	}
}

// RegisterProofreadingServiceHandler sets the in-process ProofreadingServiceHandler.
func RegisterProofreadingServiceHandler(h any) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	if h == nil {
		proofreadingHandler = nil
		return
	}
	if handler, ok := h.(numenv1connect.ProofreadingServiceHandler); ok {
		proofreadingHandler = handler
	}
}

// RegisterMonolithPluginHandler sets a dedicated handler for a plugin ID.
func RegisterMonolithPluginHandler(pluginID string, h http.Handler) {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	monolithRegistry[pluginID] = h
}

// ClearMonolithHandlers removes all registered in-process handlers.
func ClearMonolithHandlers() {
	monolithMu.Lock()
	defer monolithMu.Unlock()
	clear(monolithRegistry)
	documentHandler = nil
	recognitionHandler = nil
	bookHandler = nil
	transcriptionHandler = nil
	embeddingHandler = nil
	downloadHandler = nil
	agentHandler = nil
	proofreadingHandler = nil
}

type pipeResponseWriter struct {
	header      http.Header
	code        int
	headerReady chan struct{}
	once        sync.Once
	pw          *io.PipeWriter
}

func newPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	return &pipeResponseWriter{
		header:      make(http.Header),
		code:        http.StatusOK,
		headerReady: make(chan struct{}),
		pw:          pw,
	}
}

func (w *pipeResponseWriter) Header() http.Header {
	return w.header
}

func (w *pipeResponseWriter) Write(b []byte) (int, error) {
	w.once.Do(func() {
		close(w.headerReady)
	})
	return w.pw.Write(b)
}

func (w *pipeResponseWriter) WriteHeader(code int) {
	w.code = code
	w.once.Do(func() {
		close(w.headerReady)
	})
}

// InProcessTransport dispatches HTTP requests directly to an in-memory handler.
type InProcessTransport struct {
	handler http.Handler
}

// RoundTripTransport marks InProcessTransport as satisfying port.PluginTransport.
func (t *InProcessTransport) RoundTripTransport() {}

// NewInProcessTransport wraps an HTTP handler in an in-memory round tripper.
func NewInProcessTransport(handler http.Handler) *InProcessTransport {
	return &InProcessTransport{handler: handler}
}

// RoundTrip executes the HTTP request in-process without network overhead.
func (t *InProcessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	pr, pw := io.Pipe()
	rw := newPipeResponseWriter(pw)

	go func() {
		defer func() {
			rw.once.Do(func() {
				close(rw.headerReady)
			})
			if r := recover(); r != nil {
				_ = pw.CloseWithError(fmt.Errorf("panic in in-process handler: %v", r))
			} else {
				_ = pw.Close()
			}
		}()
		t.handler.ServeHTTP(rw, req)
	}()

	select {
	case <-rw.headerReady:
	case <-req.Context().Done():
		_ = pw.CloseWithError(req.Context().Err())
		return nil, req.Context().Err()
	}

	resp := &http.Response{
		StatusCode: rw.code,
		Status:     fmt.Sprintf("%d %s", rw.code, http.StatusText(rw.code)),
		Header:     rw.header,
		Body:       pr,
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return resp, nil
}

type monolithLifecycleServer struct {
	numenv1connect.UnimplementedPluginLifecycleServiceHandler
}

func (s *monolithLifecycleServer) Ping(
	_ context.Context,
	_ *connect.Request[v1.PingRequest],
) (*connect.Response[v1.PingResponse], error) {
	return connect.NewResponse(&v1.PingResponse{
		Status: "ok",
	}), nil
}

func (s *monolithLifecycleServer) Shutdown(
	_ context.Context,
	_ *connect.Request[v1.ShutdownRequest],
) (*connect.Response[v1.ShutdownResponse], error) {
	return connect.NewResponse(&v1.ShutdownResponse{
		Success: true,
	}), nil
}

func (s *monolithLifecycleServer) Configure(
	_ context.Context,
	_ *connect.Request[v1.ConfigureRequest],
) (*connect.Response[v1.ConfigureResponse], error) {
	return connect.NewResponse(&v1.ConfigureResponse{}), nil
}

func (h *Host) activatePlugin(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	h.mu.RLock()
	launcher := h.launcher
	h.mu.RUnlock()
	if launcher != nil {
		return h.startPluginProcess(ctx, p, sup)
	}

	if err := h.exportRuntimeVars(ctx, p.Manifest.Runtimes); err != nil {
		return emptyEndpoint(), fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}
	handler := h.getMonolithHandler(p.Manifest.ID)
	transport := NewInProcessTransport(handler)

	endpoint := Endpoint{
		Host:      "inprocess",
		Port:      0,
		Token:     "monolith",
		Transport: transport,
	}

	now := time.Now()
	h.mu.Lock()
	if sup.GetState() != StateActivating {
		h.mu.Unlock()
		return emptyEndpoint(), errActivationAborted
	}
	h.endpoints[p.Manifest.ID] = endpoint
	sup.ActivateInProcess(now)
	h.mu.Unlock()

	return endpoint, nil
}

// exportRuntimeVars provisions the runtimes an in-process plugin declares and
// sets their variables in this process, which is the plugin's own.
func (h *Host) exportRuntimeVars(ctx context.Context, runtimes []string) error {
	vars, err := h.getRuntimeVars(ctx, runtimes)
	if err != nil {
		return err
	}
	for _, entry := range vars {
		name, value, _ := strings.Cut(entry, "=")
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}

func (h *Host) getMonolithHandler(pluginID string) http.Handler {
	monolithMu.RLock()
	defer monolithMu.RUnlock()

	if custom, ok := monolithRegistry[pluginID]; ok && custom != nil {
		return custom
	}

	mux := http.NewServeMux()
	life := &monolithLifecycleServer{}
	mux.Handle(numenv1connect.NewPluginLifecycleServiceHandler(life))

	if documentHandler != nil {
		mux.Handle(numenv1connect.NewDocumentServiceHandler(documentHandler))
	}
	if recognitionHandler != nil {
		mux.Handle(numenv1connect.NewRecognitionServiceHandler(recognitionHandler))
	}
	if bookHandler != nil {
		mux.Handle(numenv1connect.NewBookServiceHandler(bookHandler))
	}
	if transcriptionHandler != nil {
		mux.Handle(numenv1connect.NewTranscriptionServiceHandler(transcriptionHandler))
	}
	if embeddingHandler != nil {
		mux.Handle(numenv1connect.NewEmbeddingServiceHandler(embeddingHandler))
	}
	if downloadHandler != nil {
		mux.Handle(numenv1connect.NewDownloadServiceHandler(downloadHandler))
	}
	if agentHandler != nil {
		mux.Handle(numenv1connect.NewAgentServiceHandler(agentHandler))
	}
	if proofreadingHandler != nil {
		mux.Handle(numenv1connect.NewProofreadingServiceHandler(proofreadingHandler))
	}

	return mux
}

var builtInMonolithPlugins = map[string]DiscoveredPlugin{
	"md.numen.pdf": {
		Dir: "modules/plugins/reader-pdf",
		Manifest: Manifest{
			ID:           "md.numen.pdf",
			Name:         "PDF Reader",
			Version:      "1.0.0",
			Capabilities: []string{"document"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.DocumentService",
			},
		},
	},
	"md.numen.ocr": {
		Dir: "modules/plugins/ocr",
		Manifest: Manifest{
			ID:           "md.numen.ocr",
			Name:         "Optical Character Recognition",
			Version:      "1.0.0",
			Capabilities: []string{"recognition"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Runtimes:     []string{port.RuntimeONNX},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.RecognitionService",
			},
		},
	},
	"md.numen.epub": {
		Dir: "modules/plugins/reader-epub",
		Manifest: Manifest{
			ID:           "md.numen.epub",
			Name:         "EPUB Reader",
			Version:      "1.0.0",
			Capabilities: []string{"book"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.BookService",
			},
		},
	},
	"md.numen.transcription": {
		Dir: "modules/plugins/transcription",
		Manifest: Manifest{
			ID:           "md.numen.transcription",
			Name:         "Audio Transcription",
			Version:      "1.0.0",
			Capabilities: []string{"transcription"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Runtimes:     []string{port.RuntimeONNX},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.TranscriptionService",
			},
		},
	},
	"md.numen.embeddings": {
		Dir: "modules/plugins/embeddings",
		Manifest: Manifest{
			ID:           "md.numen.embeddings",
			Name:         "Vector Embeddings",
			Version:      "1.0.0",
			Capabilities: []string{"embedding"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Runtimes:     []string{port.RuntimeONNX},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.EmbeddingService",
			},
		},
	},
	"md.numen.downloader": {
		Dir: "modules/plugins/downloader",
		Manifest: Manifest{
			ID:           "md.numen.downloader",
			Name:         "Media Downloader",
			Version:      "1.0.0",
			Capabilities: []string{"downloader", "media.download"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionNetworkOutbound, PermissionVaultWrite},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.DownloadService",
			},
		},
	},
	"md.numen.agent": {
		Dir: "modules/plugins/agent",
		Manifest: Manifest{
			ID:           "md.numen.agent",
			Name:         "Autonomous Agent & Chat Provider",
			Version:      "1.0.0",
			Capabilities: []string{"agent"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead, PermissionVaultWrite},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.AgentService",
			},
		},
	},
	"md.numen.proofreading": {
		Dir: "modules/plugins/proofreading",
		Manifest: Manifest{
			ID:           "md.numen.proofreading",
			Name:         "LLM Proofreading",
			Version:      "1.0.0",
			Capabilities: []string{"proofreading"},
			Targets:      []string{"desktop.editor"},
			Permissions:  []string{PermissionVaultRead},
			Backend: Backend{
				Type:        "process",
				ServiceName: "numen.v1.ProofreadingService",
			},
		},
	},
}

func (h *Host) resolveMonolithPlugin(pluginID string) (DiscoveredPlugin, bool) {
	p, ok := builtInMonolithPlugins[pluginID]
	return p, ok
}
