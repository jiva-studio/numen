package windowsession

import "context"

// RebuildSession builds the passes behind the open vault again, for a window whose notes must be cut
// again.
type RebuildSession struct {
	life *lifecycle
}

// Execute stops the passes behind the vault and starts them again. A window that shows no vault or has
// closed is left as it is.
func (u RebuildSession) Execute() {
	rt := u.life.rt
	rt.lock.Lock()
	defer rt.lock.Unlock()
	current := rt.GetCurrentVault()
	if current.ID == "" || rt.guard.isClosed() {
		return
	}
	u.life.stopVault()
	u.life.resetStatus()
	u.life.startVault(context.Background(), current, false)
}
