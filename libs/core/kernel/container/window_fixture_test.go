package container_test

import (
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	vaultregistry "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/registry"
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
	cfg      container.Config
	registry vaultregistry.Ports
	db       *vault.Index
}

func (w windowVaults) GetRefresh() vault.Refresh { return w.cfg.GetRefresh(w.db) }

func (w windowVaults) OpenNotes(index edit.IndexUpdater) note.Scenarios {
	return w.cfg.Notes(w.db, index)
}

func (w windowVaults) OpenPluginVault(
	current func() (domain.Vault, bool), hooks container.PluginVaultHooks,
) (*hostendpoint.VaultAccess, error) {
	if err := w.cfg.InstallPluginVault(w.registry, w.db, current, hooks); err != nil {
		return nil, err
	}
	return w.cfg.PluginHost.GetPluginVault(), nil
}
