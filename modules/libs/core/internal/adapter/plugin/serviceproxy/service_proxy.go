package serviceproxy

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// Plugins finds the plugin that serves a service to the windows and activates it.
type Plugins interface {
	pluginport.PluginResolver
	pluginport.PluginActivator
}

// Proxy forwards the calls a window receives for the services a plugin serves, with the
// token the plugin knows the host by. It holds a reference on the plugin from the first byte of a
// call to the last, so a stream outlives no plugin.
type Proxy struct {
	host Plugins
}

// Open makes the proxy for the plugins the host runs. It starts none.
func Open(host Plugins) *Proxy {
	return &Proxy{host: host}
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
	_, found := p.host.ResolveService(svc)
	return found
}

// hopHeaders are the headers of one connection and not of the call.
var hopHeaders = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// ServeHTTP forwards the call to the plugin that serves its service.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	found, serves := p.host.ResolveService(GetService(r.URL.Path))
	if !serves {
		writeConnectError(w, http.StatusNotFound, connect.CodeNotFound, "no plugin serves this service")
		return
	}
	endpoint, err := p.host.EnsureActive(r.Context(), found.Manifest.ID)
	if err != nil {
		code := connect.CodeUnavailable
		if errors.Is(err, context.Canceled) {
			code = connect.CodeCanceled
		}
		writeConnectError(w, http.StatusServiceUnavailable, code, bridge.ConvertActivationError(found.Manifest.ID, err).Error())
		return
	}
	lease := &bridge.Lease{Host: p.host, PluginID: found.Manifest.ID}
	defer func() { _ = lease.Close() }()

	out, err := http.NewRequestWithContext(r.Context(), r.Method, endpoint.BaseURL()+r.URL.RequestURI(), r.Body)
	if err != nil {
		writeConnectError(w, http.StatusBadRequest, connect.CodeInvalidArgument, err.Error())
		return
	}
	copyHeaders(out.Header, r.Header)
	out.Header.Del("Authorization")
	out.Header.Del("Origin")
	out.Header.Set("Authorization", "Bearer "+endpoint.Token)
	out.ContentLength = r.ContentLength

	res, err := transport.EndpointClient(endpoint).Do(out)
	if err != nil {
		writeConnectError(w, http.StatusBadGateway, connect.CodeUnavailable, err.Error())
		return
	}
	defer res.Body.Close()
	copyHeaders(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	streamBody(w, res.Body)
}

func copyHeaders(to, from http.Header) {
	for name, values := range from {
		to[name] = append([]string(nil), values...)
	}
	for _, name := range hopHeaders {
		to.Del(name)
	}
}

// streamBody copies the body as it arrives, flushing after each read so that a stream reaches the
// window message by message.
func streamBody(w http.ResponseWriter, body io.Reader) {
	flusher := http.NewResponseController(w)
	buf := make([]byte, 32*1024)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			_ = flusher.Flush()
		}
		if err != nil {
			return
		}
	}
}

// writeConnectError answers a call that did not reach a plugin as Connect clients read an error.
func writeConnectError(w http.ResponseWriter, status int, code connect.Code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code.String(), "message": message})
}
