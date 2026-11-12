package link_test

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func vaultsOf(v *hosttest.Vault) func(domain.VaultID) link.Vault {
	return func(id domain.VaultID) link.Vault {
		on := v.With(id)
		return link.Vault{Files: on, Writer: on}
	}
}

func foldersOf(v *hosttest.Vault) func(domain.VaultID) link.Folder {
	return func(id domain.VaultID) link.Folder {
		on := v.Store().With(id)
		return link.Folder{Files: on, Writer: on}
	}
}

func textsOf(texts *hosttest.Texts) func(domain.VaultID) link.Texts {
	return func(id domain.VaultID) link.Texts { return texts.With(id) }
}
