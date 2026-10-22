package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

var (
	errNotForTarget      = errors.New("the plugin does not target this host")
	errBuiltInRegistered = errors.New("a built-in plugin of this id is registered")
)

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

// Flush sends the response header, so a handler that streams is heard before it ends.
func (w *pipeResponseWriter) Flush() {
	w.once.Do(func() {
		close(w.headerReady)
	})
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
	// The body ends with the error of the request context, whether the handler has written its header or not.
	stop := context.AfterFunc(req.Context(), func() { _ = pw.CloseWithError(req.Context().Err()) })

	go func() {
		defer func() {
			stop()
			rw.once.Do(func() {
				close(rw.headerReady)
			})
			if r := recover(); r != nil {
				_ = pw.CloseWithError(fmt.Errorf("panic in in-process handler: %v", r))
			} else {
				_ = pw.CloseWithError(req.Context().Err())
			}
		}()
		t.handler.ServeHTTP(rw, req)
	}()

	select {
	case <-rw.headerReady:
	case <-req.Context().Done():
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

// RegisterBuiltIn registers a plugin linked into this binary from its manifest.json and what
// makes the handler of its backend. serve is called at each activation with the variables a
// launched process of the plugin is given: the host variables a process is launched with and the
// launch variables. A plugin of the same id found on disk is registered
// over it, and the built-in takes its place again when that one leaves the disk.
func (h *Host) RegisterBuiltIn(manifestJSON []byte, serve func(vars map[string]string) (http.Handler, error)) error {
	manifest, err := ReadManifest(bytes.NewReader(manifestJSON))
	if err != nil {
		return fmt.Errorf("register built-in plugin: %w", err)
	}
	if !slices.Contains(manifest.Targets, h.target) {
		return fmt.Errorf("register built-in plugin %s: %w: %s", manifest.ID, errNotForTarget, h.target)
	}
	p := DiscoveredPlugin{Dir: "", Manifest: manifest}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.builtIns[manifest.ID]; exists {
		return fmt.Errorf("register built-in plugin %s: %w", manifest.ID, errBuiltInRegistered)
	}
	h.builtIns[manifest.ID] = builtInPlugin{plugin: p, serve: serve}
	if _, registered := h.plugins[manifest.ID]; !registered {
		h.plugins[manifest.ID] = p
		h.registerSupervisorLocked(p)
	}
	return nil
}

func (h *Host) activatePlugin(ctx context.Context, p DiscoveredPlugin, sup *Supervisor) (Endpoint, error) {
	h.mu.RLock()
	launcher := h.launcher
	builtIn, isBuiltIn := h.getBuiltInLocked(p.Manifest.ID)
	isBuiltIn = isBuiltIn && !h.onDisk[p.Manifest.ID]
	h.mu.RUnlock()
	if launcher != nil || !isBuiltIn {
		return h.startPluginProcess(ctx, p, sup)
	}
	return h.startBuiltIn(ctx, p, builtIn, sup)
}

// startBuiltIn makes the handler of a built-in plugin and serves it in this process.
func (h *Host) startBuiltIn(ctx context.Context, p DiscoveredPlugin, builtIn builtInPlugin, sup *Supervisor) (Endpoint, error) {
	token, err := GenerateAuthToken()
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("generate token: %w", err)
	}
	runtimeVars, err := h.exportRuntimeVars(ctx, p.Manifest.GetDependencyRequirements(p.Dir))
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}
	vars := make(map[string]string)
	launchVars := h.getEnvironmentVars(ctx, p.Manifest.ID, 0, token, runtimeVars)
	for _, entry := range sanitizeEnvironment(os.Environ(), launchVars, p.Manifest.GetRuntimeIDs()) {
		name, value, _ := strings.Cut(entry, "=")
		vars[name] = value
	}
	handler, err := builtIn.serve(vars)
	if err != nil {
		return emptyEndpoint(), fmt.Errorf("launch plugin %s: %w", p.Manifest.ID, err)
	}

	endpoint := Endpoint{
		Host:      "inprocess",
		Port:      0,
		Token:     token,
		Transport: NewInProcessTransport(handler),
	}
	if err := checkPluginProtocol(ctx, p.Manifest, endpoint); err != nil {
		h.shutdownBuiltIn(ctx, endpoint)
		return emptyEndpoint(), err
	}

	now := time.Now()
	h.mu.Lock()
	if sup.GetState() != StateActivating {
		h.mu.Unlock()
		h.shutdownBuiltIn(ctx, endpoint)
		return emptyEndpoint(), errActivationAborted
	}
	h.endpoints[p.Manifest.ID] = endpoint
	sup.ActivateInProcess(now)
	h.mu.Unlock()
	return endpoint, nil
}

// shutdownBuiltIn shuts down the handler of a built-in plugin that was made and not activated, so
// what it holds open is closed.
func (h *Host) shutdownBuiltIn(ctx context.Context, endpoint Endpoint) {
	if shutdown := h.createRPCShutdown(endpoint, time.Second); shutdown != nil {
		if err := shutdown(context.WithoutCancel(ctx)); err != nil {
			h.reportError(fmt.Errorf("shut down the handler of built-in plugin that was not activated: %w", err))
		}
	}
}

// exportRuntimeVars provisions the runtimes an in-process plugin declares, sets their variables in
// this process, which is the plugin's own, and returns them.
func (h *Host) exportRuntimeVars(ctx context.Context, requirements []pluginport.DependencyRequirement) ([]string, error) {
	vars, err := h.getRuntimeVars(ctx, requirements)
	if err != nil {
		return nil, err
	}
	for _, entry := range vars {
		name, value, _ := strings.Cut(entry, "=")
		if err := os.Setenv(name, value); err != nil {
			return nil, err
		}
	}
	return vars, nil
}
