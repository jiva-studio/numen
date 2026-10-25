package vault

import (
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

type RecordOpen struct {
	registry vaultport.VaultRegistry
}

func NewRecordOpen(registry vaultport.VaultRegistry) RecordOpen {
	return RecordOpen{registry: registry}
}

func (u RecordOpen) Execute(id domain.VaultID) error {
	return u.registry.RecordOpened(id)
}
