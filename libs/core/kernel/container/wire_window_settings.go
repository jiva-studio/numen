package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// OpenGetSettings builds the use case that reads every setting of the installation and the models
// the settings naming a model can be set to, with the programs the provider lists.
func (c Config) OpenGetSettings(provider agentport.AgentProvider) *settings.Get {
	return settings.NewGet(c.ReadSettings(), c.modelsForProvider(provider), c.Plugins(), func() []plugin.Setting { return listAppSettings(c.settings()) })
}

// OpenWatchSettings builds the use case that reports each setting that changes.
func (c Config) OpenWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(c.settings())
}

func mountSettings(w *windowWiring) {
	state := w.inst.State
	handler := &windowsettings.Handler{}
	handler.Get = w.cfg.OpenGetSettings(w.cfg.AgentProvider())
	handler.Values.Write = openWriteSettings(w.cfg.settings())
	handler.Watch = w.cfg.OpenWatchSettings()
	handler.Retry = openRetryPlugin(w.cfg.settings(), w.cfg.GetPluginRetrier())
	if secrets := w.cfg.OpenSecrets(); secrets != nil {
		handler.ListSecrets = secrets.List
		handler.Create, handler.Rename, handler.SecretItems.Write = secrets.Create, secrets.Rename, secrets.Write
		handler.Delete, handler.Move = secrets.Delete, secrets.Move
		handler.RenameGroup, handler.DeleteGroup = secrets.RenameGroup, secrets.DeleteGroup
		handler.Allow, handler.Deny, handler.ListStates = secrets.Allow, secrets.Deny, secrets.ListStates
	}
	if data := openPluginData(w.cfg.vault()); data != nil {
		handler.Data, handler.Vaults = data, state
	}
	w.inst.OnFolderDialog(func(dialog machineport.PathDialog) {
		handler.ChoosePath = openChoosePluginPath(w.cfg.settings(), dialog)
	})
	path, served := windowsettings.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
