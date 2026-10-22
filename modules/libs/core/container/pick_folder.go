package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
)

func (c Config) PickFolder(dialog machineport.FolderDialog) *vault.PickFolder {
	if dialog == nil {
		return nil
	}
	pick := vault.NewPickFolder(dialog)
	return &pick
}
