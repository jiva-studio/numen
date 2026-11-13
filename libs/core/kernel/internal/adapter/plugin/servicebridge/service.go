package servicebridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Plugins finds the plugin that serves a service to the windows, and lists the plugins installed.
type Plugins interface {
	ResolveService(service string) (manifest.Discovered, bool)
	GetPlugin(id string) (manifest.Discovered, bool)
	GetPluginInfos() []plugin.Info
}

// Proxy forwards the calls a window receives for the services a plugin serves, with the
// token by which the plugin identifies the host. It holds a reference on the plugin from the first byte of a
// call to the last, so a stream cannot outlive its plugin.
type Proxy struct {
	activator bridge.Activator
	plugins   Plugins
	onError   func(error)
}

// SetErrorHandler sets the callback that receives errors the proxy meets after a response has begun.
// It must be called before the proxy serves.
func (p *Proxy) SetErrorHandler(handler func(error)) {
	p.onError = handler
}

func (p *Proxy) reportError(err error) {
	if err != nil && p.onError != nil {
		p.onError(err)
	}
}

func (p *Proxy) closeResponse(res *http.Response) {
	if err := res.Body.Close(); err != nil {
		p.reportError(fmt.Errorf("close plugin response: %w", err))
	}
}

func (p *Proxy) releasePlugin(lease *bridge.Lease) {
	if err := lease.Close(); err != nil {
		p.reportError(fmt.Errorf("release plugin %s: %w", lease.PluginID, err))
	}
}

// Open makes the proxy for the plugins that activator starts. It starts none.
func Open(activator bridge.Activator, plugins Plugins) *Proxy {
	return &Proxy{activator: activator, plugins: plugins}
}

// GetService is the service a request path names: the first segment of /Service/Method.
func GetService(urlPath string) string {
	service, _, _ := strings.Cut(strings.TrimPrefix(urlPath, "/"), "/")
	return service
}

// Serves reports whether an enabled plugin serves the service the path names.
func (p *Proxy) Serves(urlPath string) bool {
	svc := GetService(urlPath)
	if svc == "" {
		return false
	}
	_, found := p.plugins.ResolveService(svc)
	return found
}

func getHopHeaders() []string {
	return []string{
		"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
		"Te", "Trailer", "Transfer-Encoding", "Upgrade",
	}
}

// ServeHTTP forwards the call to the plugin that serves its service.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	found, serves := p.plugins.ResolveService(GetService(r.URL.Path))
	if !serves {
		p.reportError(writeConnectError(w, http.StatusNotFound, connect.CodeNotFound, "no plugin serves this service"))
		return
	}
	holder := plugin.GetHolder(plugin.WithAskingPlugin(r.Context(), found.Manifest.ID), plugin.GetCallTitle(r.URL.Path))
	endpoint, err := p.activator.EnsureActive(r.Context(), found.Manifest.ID, holder)
	if err != nil {
		code := connect.CodeUnavailable
		if errors.Is(err, context.Canceled) {
			code = connect.CodeCanceled
		}
		p.reportError(writeConnectError(w, http.StatusServiceUnavailable, code, bridge.ConvertActivationError(found.Manifest.ID, err).Error()))
		return
	}
	lease := &bridge.Lease{Host: p.activator, PluginID: found.Manifest.ID, Token: endpoint.Token, Holder: holder}
	defer p.releasePlugin(lease)

	out, err := http.NewRequestWithContext(r.Context(), r.Method, transport.GetBaseURL(endpoint)+r.URL.RequestURI(), r.Body)
	if err != nil {
		p.reportError(writeConnectError(w, http.StatusBadRequest, connect.CodeInvalidArgument, err.Error()))
		return
	}
	copyHeaders(out.Header, r.Header)
	out.Header.Del("Authorization")
	out.Header.Del("Origin")
	out.Header.Set("Authorization", "Bearer "+endpoint.Token)
	out.ContentLength = r.ContentLength

	res, err := transport.GetClient(p.activator, endpoint).Do(out)
	if err != nil {
		p.reportError(writeConnectError(w, http.StatusBadGateway, connect.CodeUnavailable, err.Error()))
		return
	}
	defer p.closeResponse(res)
	copyHeaders(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	p.reportError(streamBody(w, res.Body))
}

func copyHeaders(to, from http.Header) {
	for name, values := range from {
		to[name] = append([]string(nil), values...)
	}
	for _, name := range getHopHeaders() {
		to.Del(name)
	}
}

func streamBody(w http.ResponseWriter, body io.Reader) error {
	flusher := http.NewResponseController(w)
	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write response: %w", werr)
			}
			if ferr := flusher.Flush(); ferr != nil {
				return fmt.Errorf("flush response: %w", ferr)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read plugin response: %w", err)
		}
	}
}

func writeConnectError(w http.ResponseWriter, status int, code connect.Code, message string) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"code": code.String(), "message": message}); err != nil {
		return fmt.Errorf("write error response: %w", err)
	}
	return nil
}
