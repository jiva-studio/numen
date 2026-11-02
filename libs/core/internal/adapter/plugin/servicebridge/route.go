package servicebridge

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
)

// ServeRoute forwards a GET or HEAD request on the plugin route to the backend of the plugin it
// names and streams the response back unread. The plugin sets no Content-Security-Policy or cookie.
func (p *Proxy) ServeRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, rest, ok := parseRoute(r.URL.Path)
	if !ok {
		http.Error(w, "invalid plugin route", http.StatusBadRequest)
		return
	}
	if _, enabled := p.host.GetPlugin(id); !enabled {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	endpoint, err := p.host.EnsureActive(r.Context(), id)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, context.Canceled) {
			status = http.StatusRequestTimeout
		}
		http.Error(w, bridge.ConvertActivationError(id, err).Error(), status)
		return
	}
	lease := &bridge.Lease{Host: p.host, PluginID: id}
	defer p.releasePlugin(lease)

	target := url.URL{Path: wire.PluginRoute + rest, RawQuery: r.URL.RawQuery}
	out, err := http.NewRequestWithContext(r.Context(), r.Method, endpoint.BaseURL()+target.RequestURI(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	copyHeaders(out.Header, r.Header)
	out.Header.Del("Cookie")
	out.Header.Del("Origin")
	out.Header.Set("Authorization", "Bearer "+endpoint.Token)

	res, err := transport.EndpointClient(endpoint).Do(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer p.closeResponse(res)
	// The window sets its own policy before forwarding; the plugin cannot widen it.
	res.Header.Del("Content-Security-Policy")
	res.Header.Del("Set-Cookie")
	copyHeaders(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	p.reportError(streamBody(w, res.Body))
}

// parseRoute splits a path of the plugin route into the plugin id and the path below it. A path
// whose rest climbs out of the route, or is not in its clean form, is refused.
func parseRoute(urlPath string) (id, rest string, ok bool) {
	tail, found := strings.CutPrefix(urlPath, wire.PluginRoute)
	if !found {
		return "", "", false
	}
	id, rest, found = strings.Cut(tail, "/")
	if !found || id == "" || rest == "" || strings.Contains(rest, `\`) {
		return "", "", false
	}
	if path.Clean("/"+rest) != "/"+strings.TrimSuffix(rest, "/") {
		return "", "", false
	}
	return id, rest, true
}
