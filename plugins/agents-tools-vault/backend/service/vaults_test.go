package service_test

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

func vaultsOf(v *hosttest.Vault) func(domain.VaultID) service.Vault {
	return func(id domain.VaultID) service.Vault {
		on := v.With(id)
		return service.Vault{Files: on, Writer: on, Index: on, Notes: on, Links: on, Arranger: on, Editor: on, Selector: on}
	}
}
