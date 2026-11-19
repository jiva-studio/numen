package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// PluginVaultPath points the plugins at the folder of the vault the window shows.
type PluginVaultPath interface {
	SetVaultPath(path string)
}

// SwitchVault shows a vault in the window and, where it was shown, points the plugins at it.
type SwitchVault struct {
	show     VaultSwitch
	plugins  PluginVaultPath
	tabs     *TabReports
	showTabs func(window.OpenTabs)
}

// NewSwitchVault returns the use case over the show of a vault, the plugins and the view of the open tabs.
func NewSwitchVault(show VaultSwitch, plugins PluginVaultPath, tabs *TabReports, showTabs func(window.OpenTabs)) SwitchVault {
	return SwitchVault{show: show, plugins: plugins, tabs: tabs, showTabs: showTabs}
}

// Execute returns the error of showing the vault. The view of the open tabs is shown whether or not the
// vault was shown.
func (u SwitchVault) Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	err := u.show.Execute(ctx, v, open)
	if err == nil {
		u.plugins.SetVaultPath(v.Path)
	}
	if u.showTabs != nil {
		u.showTabs(u.tabs.GetOpenTabs())
	}
	return err
}
