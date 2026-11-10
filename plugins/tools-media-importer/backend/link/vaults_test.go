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

// foldersOf gives the plugin folder in the store of the vault with an identity, seen from the vault.
func foldersOf(v *hosttest.Vault) func(domain.VaultID) link.Folder {
	return func(id domain.VaultID) link.Folder {
		on := v.Store().With(id)
		return link.Folder{Files: on, Writer: on}
	}
}

// textsOf gives the text versions of the vault with an identity.
func textsOf(texts *hosttest.Texts) func(domain.VaultID) link.Texts {
	return func(id domain.VaultID) link.Texts { return texts.With(id) }
}
