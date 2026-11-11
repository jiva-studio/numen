package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginVaultPath points the plugins at the folder of the vault the window shows.
type PluginVaultPath interface {
	SetVaultPath(path string)
}

// OpenTabsView shows the tabs open in the window after a switch.
type OpenTabsView interface {
	ShowOpenTabs()
}

// SwitchVault shows a vault in the window and, where it was shown, points the plugins at it.
type SwitchVault struct {
	show    ShowVault
	plugins PluginVaultPath
	tabs    OpenTabsView
}

// NewSwitchVault returns the use case over the show of a vault, the plugins and the view of the open tabs.
func NewSwitchVault(show ShowVault, plugins PluginVaultPath, tabs OpenTabsView) SwitchVault {
	return SwitchVault{show: show, plugins: plugins, tabs: tabs}
}

// Execute returns the error of showing the vault. The view of the open tabs is shown whether or not the
// vault was shown.
func (u SwitchVault) Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	err := u.show.Execute(ctx, v, open)
	if err == nil {
		u.plugins.SetVaultPath(v.Path)
	}
	u.tabs.ShowOpenTabs()
	return err
}
