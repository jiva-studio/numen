package container_test

import (
	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// editorServices is what the editor window is given of a configuration: its agent and the use cases
// of its settings. The agent and its provider are made once, as a window makes them.
type editorServices struct {
	cfg      container.Config
	agent    agentport.Agent
	provider agentport.AgentProvider
}

func newEditorServices(cfg container.Config) editorServices {
	return editorServices{cfg: cfg, agent: cfg.GetEditorAgent(), provider: cfg.AgentProvider()}
}

func (e editorServices) GetAgent() agentport.Agent { return e.agent }

func (e editorServices) OpenGetSettings() *settings.Get { return e.cfg.OpenGetSettings(e.provider) }

func (e editorServices) OpenWatchSettings() *settings.WatchSettingChanges {
	return e.cfg.OpenWatchSettings()
}

// windowVaults is a configuration with the list of vaults and the cache a window works the vaults
// of the plugins through.
type windowVaults struct {
	cfg container.Config
	db  *container.Index
}

func (w windowVaults) GetRefresh() vault.Refresh { return container.GetRefresh(w.cfg, w.db) }

func (w windowVaults) OpenNotes(index edit.IndexUpdater) note.Scenarios {
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
) note.Scenarios {
	return container.NotesShowingMoves(w.cfg, w.db, index, showing, view)
}
