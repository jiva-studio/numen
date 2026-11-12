package editor

import (
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalogtest"

	windowcatalog "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/catalog"
)

// A route with nothing mounted falls through to the pages and answers not found; a route a service
// owns answers a GET with a method-not-allowed error.
func TestCatalogServiceIsMountedOnlyWhereTheInstallationHasACatalog(t *testing.T) {
	route := numenv1connect.CatalogServiceListCatalogProcedure
	service, err := catalog.NewService(catalog.Config{
		Target: "desktop.editor", Sources: []catalog.Source{catalogtest.NewFakeSource("local")}, InstallDir: "", CacheDir: "",
	})
	if err != nil {
		t.Fatal(err)
	}

	noCatalog := newBareServices().NewHandler(http.NotFoundHandler())
	services := newBareServices()
	services.Mount(RouteOf(services.state, windowcatalog.NewHandler(service), windowcatalog.NewRoute))
	withCatalog := services.NewHandler(http.NotFoundHandler())

	if code := getResponse(t, noCatalog, route).Code; code != http.StatusNotFound {
		t.Errorf("%s answered %d without a catalog, want %d", route, code, http.StatusNotFound)
	}
	if code := getResponse(t, withCatalog, route).Code; code != http.StatusMethodNotAllowed {
		t.Errorf("%s answered %d with a catalog, want %d", route, code, http.StatusMethodNotAllowed)
	}
}
