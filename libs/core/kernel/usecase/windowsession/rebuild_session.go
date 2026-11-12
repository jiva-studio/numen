package windowsession

import "context"

// RebuildSession builds the passes behind the open vault again, for a window whose notes must be cut
// again.
type RebuildSession struct {
	guard   *Guard
	session vaultSession
}

// NewRebuildSession makes the use case over the guard and the session of the window.
func NewRebuildSession(guard *Guard, session *Session) RebuildSession {
	return RebuildSession{guard: guard, session: session}
}

// Execute stops the passes behind the vault and starts them again. A window that shows no vault or has
// closed is left as it is.
func (u RebuildSession) Execute(ctx context.Context) {
	u.guard.switching.Lock()
	defer u.guard.switching.Unlock()
	current := u.session.getCurrentVault()
	if current.ID == "" || u.guard.isClosed() {
		return
	}
	u.session.stopVault()
	u.session.resetStatus()
	u.session.startVault(ctx, current, false)
}
