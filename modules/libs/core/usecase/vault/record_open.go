package vault

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

type RecordOpen struct {
	registry port.VaultRegistry
}

func NewRecordOpen(registry port.VaultRegistry) RecordOpen {
	return RecordOpen{registry: registry}
}

func (u RecordOpen) Execute(id domain.VaultID) error {
	return u.registry.RecordOpened(id)
}
