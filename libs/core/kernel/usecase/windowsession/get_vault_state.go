package windowsession

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// GetVaultState reads what the window knows about the vault it shows: whether the first scan is done and
// what stopped it.
func GetVaultState(rt *Runtime) window.VaultState {
	return window.VaultState{
		Vault:      rt.GetCurrentVault(),
		IsReady:    rt.IsReady(),
		Error:      rt.GetError(),
		WatchError: rt.GetWatchError(),
	}
}
