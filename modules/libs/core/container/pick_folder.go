package container

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c Config) PickFolder(dialog port.FolderDialog) *vault.PickFolder {
	if dialog == nil {
		return nil
	}
	return vault.NewPickFolder(dialog)
}
