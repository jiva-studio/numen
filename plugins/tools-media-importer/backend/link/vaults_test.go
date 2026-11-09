package link_test

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// vaultsOf gives the calls on the vault of v with an identity; the empty identity is v itself.
func vaultsOf(v *hosttest.Vault) func(domain.VaultID) link.Vault {
	return func(id domain.VaultID) link.Vault {
		on := v.With(id)
		return link.Vault{Files: on, Writer: on}
	}
}
