package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	editorsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// OpenGetSettings builds the use case that reads every setting of the installation and the models
// the settings naming a model can be set to, with the programs the provider lists.
func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settings.Get {
	return settings.NewGet(c.ReadSettings(), c.modelsForProvider(provider), c.Plugins(), c.settings().ListAppSettings)
}

// OpenWatchSettings builds the use case that reports each setting that changes.
func (c Config) OpenWatchSettings() *settings.Watch {
	return c.settings().OpenWatchSettings()
}

// mountSettings mounts the service that reads and writes the settings of the installation.
func mountSettings(w *windowWiring) {
	state := w.inst.State
	handler := editorsettings.NewHandler()
	handler.Get = w.cfg.OpenGetSettings(w.cfg.AgentProvider())
	handler.Write = w.cfg.settings().OpenWriteSettings()
	handler.Watch = w.cfg.OpenWatchSettings()
	handler.Secrets = w.cfg.OpenSecrets()
	handler.PluginRetry = w.cfg.settings().OpenRetryPlugin(w.cfg.GetPluginRetrier())
	if data := w.cfg.vault().OpenPluginData(); data != nil {
		handler.PluginData, handler.Vaults = data, state
	}
	w.inst.OnFolderDialog(func(dialog machineport.PathDialog) {
		handler.ChoosePath = w.cfg.settings().OpenChoosePluginPath(dialog)
	})
	w.inst.Mount(editor.NewSettingsRoute(state, handler))
}
