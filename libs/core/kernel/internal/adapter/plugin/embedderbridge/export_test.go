package embedderbridge

// WaitForAnnouncements returns once no announcement of a model change is running.
func (b *Embedder) WaitForAnnouncements() {
	b.announceMu.Lock()
	defer b.announceMu.Unlock()
}

// IsListenerLockFree reports whether the embedder holds the lock that guards its listeners.
func (b *Embedder) IsListenerLockFree() bool {
	if !b.changeMu.TryLock() {
		return false
	}
	b.changeMu.Unlock()
	return true
}
