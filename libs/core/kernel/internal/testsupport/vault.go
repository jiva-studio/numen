// Package testsupport gives tests the things they all need and none of them
// should spell out.
package testsupport

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// SaveVaults puts the vaults on the list of vaults kept at registryPath.
func SaveVaults(tb testing.TB, registryPath string, vaults ...domain.Vault) {
	tb.Helper()
	list := appstate.OpenAt(registryPath)
	for _, v := range vaults {
		if err := list.Save(v); err != nil {
			tb.Fatal(err)
		}
	}
}
