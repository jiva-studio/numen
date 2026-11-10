package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowsettings "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func (c Config) openGetSettings(provider agentport.AgentProvider) *settings.Get {
	return settings.NewGet(c.readSettings(), c.modelsForProvider(provider), c.Plugins(), func() []plugin.Setting { return listAppSettings(c.settings()) })
}

func (c Config) openWatchSettings() *settings.WatchSettingChanges {
	return openWatchSettings(c.settings())
}

func mountSettings(w *windowWiring) {
	state := w.inst.State
	handler := &windowsettings.Handler{}
	handler.Get = w.cfg.openGetSettings(w.cfg.agentProvider())
	handler.Values.Write = openWriteSettings(w.cfg.settings())
	handler.Watch = w.cfg.openWatchSettings()
	handler.Retry = openRetryPlugin(w.cfg.settings(), w.cfg.getPluginRetrier())
	if secrets := w.cfg.openSecrets(); secrets != nil {
		handler.ListSecrets = secrets.List
		handler.Create, handler.Rename, handler.SecretItems.Write = secrets.Items.Create, secrets.Items.Rename, secrets.Items.Write
		handler.Delete, handler.Move = secrets.Items.Delete, secrets.Items.Move
		handler.RenameGroup, handler.DeleteGroup = secrets.Groups.Rename, secrets.Groups.Delete
		handler.Allow, handler.Deny, handler.ListStates = secrets.Access.Allow, secrets.Access.Deny, secrets.Access.ListStates
	}
	if data := openPluginData(w.cfg.vault()); data != nil {
		handler.Data, handler.Vaults = data, state
	}
	w.inst.OnFolderDialog(func(dialog editor.PathDialog) {
		handler.ChoosePath = openChoosePluginPath(w.cfg.settings(), dialog)
	})
	path, served := windowsettings.NewRoute(handler, state.CountQuestions())
	w.inst.Mount(editor.NewRoute(path, served, handler))
}
