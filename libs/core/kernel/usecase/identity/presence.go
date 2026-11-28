package identity

import "context"

// IsPresent reports whether this device has an identity, or is on the way to one: a restore waits for the chain of the
// server, or a link to an account is in progress. A key store that cannot be read counts as holding an identity.
func (a *Account) IsPresent(ctx context.Context) bool {
	store := a.keys()
	if store == nil {
		return false
	}
	_, err := store.GetKey(ctx, keyName)
	if err == nil || !isNoKey(err) {
		return true
	}
	if a.progress.pending.read() != nil {
		return true
	}
	links, release := a.progress.links.hold()
	defer release()
	return links.current != nil && !links.current.isEnded()
}

// OnChange sets a function that is called when IsPresent may have changed. It returns at once and calls no use case.
func (a *Account) OnChange(listener func()) { a.progress.OnChange(listener) }
