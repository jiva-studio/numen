package editor

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/csp"
)

func (s *Services) getPolicy(played []string) string {
	frames := append([]string(nil), played...)
	if s.PluginServices != nil {
		frames = append(frames, s.PluginServices.ListFrameOrigins()...)
	}
	return csp.Sources{Media: played, Frames: frames}.Policy()
}

func pluginPagePolicy(frames []string) string {
	return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"frame-src " + strings.Join(append([]string{"'self'"}, frames...), " ") +
		"; object-src 'none'; base-uri 'none'; form-action 'none'"
}

// pluginPageReferrer is the Referrer-Policy of a plugin's page on the loopback socket. A site that
// checks which page frames its player is sent the origin of the socket.
const pluginPageReferrer = "strict-origin-when-cross-origin"

func (l *Loopback) servePluginPage(w http.ResponseWriter, r *http.Request, rest string) {
	plugins := l.getPluginServices()
	if plugins == nil {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	path, err := url.PathUnescape(rest)
	if err != nil {
		http.Error(w, "invalid plugin route", http.StatusBadRequest)
		return
	}
	id, _, _ := strings.Cut(path, "/")
	forwarded := r.Clone(r.Context())
	forwarded.URL.Path = wire.PluginRoute + path
	forwarded.URL.RawPath = ""
	w.Header().Set("Content-Security-Policy", pluginPagePolicy(plugins.GetFrameOrigins(id)))
	w.Header().Set("Referrer-Policy", pluginPageReferrer)
	plugins.ServeRoute(w, forwarded)
}

// PageAddress is where a page the plugin serves at path below its route is loaded from on the
// loopback socket, and nowhere where the socket is not answering.
func (l *Loopback) PageAddress(pluginID, path string) string {
	if l == nil || l.stopped.Load() {
		return ""
	}
	return l.address + "/" + l.token + "/" + pluginPageSegment + "/" + pluginID + "/" + strings.TrimPrefix(path, "/")
}

// pluginPageSegment is the first segment below the token of a plugin's page. It names no vault.
const pluginPageSegment = "plugin"

func (l *Loopback) setPluginServices(plugins PluginServices) {
	if l != nil && plugins != nil {
		l.plugins.Store(&plugins)
	}
}

func (l *Loopback) getPluginServices() PluginServices {
	if found := l.plugins.Load(); found != nil {
		return *found
	}
	return nil
}
