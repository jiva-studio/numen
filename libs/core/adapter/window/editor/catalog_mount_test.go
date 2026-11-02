package editor

import (
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// A route with nothing mounted falls through to the pages and answers not found; a route a service
// owns answers a GET with a method-not-allowed error.
func TestCatalogServiceIsMountedOnlyWhereTheInstallationHasACatalog(t *testing.T) {
	route := numenv1connect.CatalogServiceListCatalogProcedure

	noCatalog := newBareServices().NewHandler(http.NotFoundHandler())
	services := newBareServices()
	services.Catalog = numenv1connect.UnimplementedCatalogServiceHandler{}
	withCatalog := services.NewHandler(http.NotFoundHandler())

	if code := getResponse(t, noCatalog, route).Code; code != http.StatusNotFound {
		t.Errorf("%s answered %d without a catalog, want %d", route, code, http.StatusNotFound)
	}
	if code := getResponse(t, withCatalog, route).Code; code != http.StatusMethodNotAllowed {
		t.Errorf("%s answered %d with a catalog, want %d", route, code, http.StatusMethodNotAllowed)
	}
}
