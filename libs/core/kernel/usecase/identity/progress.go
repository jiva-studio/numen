package identity

// Progress keeps in memory what the identity use cases have in progress: the restore that waits for the chain of a
// sync plugin, and the link and the revoke that wait for the other device and the server. The use cases of the
// windows and of the plugins of one installation share one.
type Progress struct {
	restores *pendingStore
	links    linkHold
}

// NewProgress returns a progress with no restore, no link and no revoke.
func NewProgress() *Progress {
	return &Progress{restores: newPendingStore(), links: newLinkStore()}
}
