package container_test

import (
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// windowVaults is a configuration with the list of vaults and the cache a window works the vaults
// of the plugins through.
type windowVaults struct {
	cfg container.Config
	db  *container.Index
}

func (w windowVaults) GetRefresh() vault.Refresh { return container.GetRefresh(w.cfg, w.db) }

func (w windowVaults) OpenNotes(index edit.IndexUpdater) container.NoteUseCases {
	return container.Notes(w.cfg, w.db, index)
}

func (w windowVaults) OpenPluginVault(
	current func() (domain.Vault, bool), hooks container.PluginVaultHooks,
) (*vaultapi.Access, error) {
	if err := container.InstallPluginVault(w.cfg, w.db, current, hooks); err != nil {
		return nil, err
	}
	return w.cfg.PluginHost.Endpoint.GetPluginVault(), nil
}

func (w windowVaults) OpenNotesShowingMoves(
	index edit.IndexUpdater, showing func() (domain.Vault, bool), view note.MoveView,
) container.NoteUseCases {
	return container.NotesShowingMoves(w.cfg, w.db, index, showing, view)
}
