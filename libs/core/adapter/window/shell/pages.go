package shell

import (
	"embed"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/internal/csp"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

// policy is what this window may load. A plugin draws pictures it carries as `data:` URIs.
var policy = csp.Sources{Images: []string{"data:"}}.Policy()

// Pages is the interface itself, built by `make interface` and carried inside the binary. A binary
// built without it says so.
func Pages() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// NewHandler is everything this window answers: the window itself, its themes, the agent, its plugins
// and the files of their frontends, the services they serve, and the files the page is made of. The
// agent is answered here and never forwarded to a plugin that serves AgentService.
func (a *API) NewHandler(files http.Handler) http.Handler {
	windowRoute, window := numenv1connect.NewWindowServiceHandler(a.Window)
	pluginsRoute, plugins := numenv1connect.NewPluginServiceHandler(pluginList{getIDs: a.GetPluginIDs})
	agentRoute, agent := numenv1connect.NewAgentServiceHandler(&wire.AgentHandler{GetAgent: a.GetAgent})
	var themesRoute string
	var themes http.Handler
	if a.Themes != nil {
		themesRoute, themes = numenv1connect.NewThemeServiceHandler(a.Themes)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		switch {
		case strings.HasPrefix(r.URL.Path, windowRoute):
			window.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, pluginsRoute):
			plugins.ServeHTTP(w, r)
		case themes != nil && strings.HasPrefix(r.URL.Path, themesRoute):
			themes.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, agentRoute):
			agent.ServeHTTP(w, r)
		case strings.HasPrefix(r.URL.Path, wire.PluginAssetsRoute):
			wire.ServePluginAsset(w, r, a.GetPluginDir)
		case a.PluginServices != nil && a.PluginServices.Serves(r.URL.Path):
			a.PluginServices.ServeHTTP(w, r)
		case slices.Contains(wire.OpenedAt, r.URL.Path):
			wire.WritePage(w, r, pages, a.Themes, files)
		default:
			files.ServeHTTP(w, r)
		}
	})
}
