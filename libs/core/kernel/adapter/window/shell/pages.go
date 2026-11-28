package shell

import (
	"embed"
	"net/http"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/csp"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

//go:embed all:pages
var pages embed.FS

func getPolicy() string { return csp.Sources{Images: []string{"data:"}}.Policy() }

// CreatePagesHandler is the interface itself, built by `make interface` and embedded in the
// binary. A binary built without it reports that.
func CreatePagesHandler() (http.Handler, error) { return wire.NewInterfaceServer(pages) }

// NewHandler is everything this window answers: the window itself, its themes, the calls its plugin
// views make through their connections, the list of vaults, where the links of a text land, its
// plugins and the files of their frontends, the services they serve, and the files the page is made
// of. A window given no ConnectionCalls, Vaults or ResolveLinks answers each such call as unimplemented.
func (a *API) NewHandler(files http.Handler) http.Handler {
	windowRoute, window := numenv1connect.NewWindowServiceHandler(a.Window)
	pluginsRoute, plugins := numenv1connect.NewPluginServiceHandler(a.Frontends)
	callsRoute, calls := numenv1connect.NewConnectionCallServiceHandler(a.getConnectionCalls())
	vaultsRoute, vaults := numenv1connect.NewVaultsServiceHandler(a.getVaults())
	notesRoute, notes := numenv1connect.NewNoteServiceHandler(a.getNotes())
	routes := []route{{windowRoute, window}, {pluginsRoute, plugins}}
	if a.Themes != nil {
		themesRoute, themes := numenv1connect.NewThemeServiceHandler(a.Themes)
		routes = append(routes, route{themesRoute, themes})
	}
	routes = append(routes, route{callsRoute, calls}, route{vaultsRoute, vaults}, route{notesRoute, notes})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", getPolicy())
		w.Header().Set("Permissions-Policy", csp.PermissionsPolicy)
		for _, rt := range routes {
			if strings.HasPrefix(r.URL.Path, rt.prefix) {
				rt.handler.ServeHTTP(w, r)
				return
			}
		}
		a.serveOther(w, r, files)
	})
}

func (a *API) getVaults() numenv1connect.VaultsServiceHandler {
	if a.Vaults == nil {
		return numenv1connect.UnimplementedVaultsServiceHandler{}
	}
	return vaultList{list: a.Vaults.Execute}
}

func (a *API) getNotes() numenv1connect.NoteServiceHandler {
	if a.ResolveLinks == nil {
		return numenv1connect.UnimplementedNoteServiceHandler{}
	}
	return noteLinks{resolve: a.ResolveLinks}
}

func (a *API) getConnectionCalls() numenv1connect.ConnectionCallServiceHandler {
	if a.ConnectionCalls == nil {
		return numenv1connect.UnimplementedConnectionCallServiceHandler{}
	}
	return a.ConnectionCalls
}

// route is a prefix of the request path and the handler that answers it.
type route struct {
	prefix  string
	handler http.Handler
}

func (a *API) serveOther(w http.ResponseWriter, r *http.Request, files http.Handler) {
	switch {
	case strings.HasPrefix(r.URL.Path, wire.PluginRoute):
		// The shell runs no plugin backend: the route serves only the files of a plugin folder.
		if !wire.ServePluginFile(w, r, a.Frontends.getPluginDir) {
			http.Error(w, "plugin file not found", http.StatusNotFound)
		}
	case a.PluginServices != nil && a.PluginServices.CanServe(r.URL.Path):
		a.PluginServices.ServeHTTP(w, r)
	case slices.Contains(wire.GetOpenedAt(), r.URL.Path):
		wire.WritePage(w, r, pages, a.Themes, files)
	default:
		files.ServeHTTP(w, r)
	}
}
