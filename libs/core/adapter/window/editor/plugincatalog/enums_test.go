package plugincatalog

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

func TestEveryCatalogStateIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CatalogState]catalog.State{
		v1.CatalogState_CATALOG_STATE_UNSPECIFIED: "",
		v1.CatalogState_CATALOG_STATE_INSTALLED:   catalog.StateInstalled,
		v1.CatalogState_CATALOG_STATE_DISABLED:    catalog.StateDisabled,
		v1.CatalogState_CATALOG_STATE_AVAILABLE:   catalog.StateAvailable,
	}, newWireState)
}

func TestEveryIncompatibilityIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CatalogIncompatibility]manifest.IncompatibleReason{
		v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_UNSPECIFIED: "",
		v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_TARGET:      manifest.ReasonTarget,
		v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_PROTOCOL:    manifest.ReasonProtocol,
		v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_APP_TOO_OLD: manifest.ReasonAppTooOld,
		v1.CatalogIncompatibility_CATALOG_INCOMPATIBILITY_APP_TOO_NEW: manifest.ReasonAppTooNew,
	}, newWireIncompatibility)
}

func TestEveryCatalogErrorCodeIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CatalogErrorCode]error{
		v1.CatalogErrorCode_CATALOG_ERROR_CODE_UNSPECIFIED: nil,
		v1.CatalogErrorCode_CATALOG_ERROR_CODE_NOT_FOUND:   catalog.ErrNotFound,
	}, newWireErrorCode)
}
