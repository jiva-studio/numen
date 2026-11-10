package vault

import "sync/atomic"

// CurrentRefresh is the refresh of the vault being opened, which a note saved by the user goes through so
// the index reflects it before the save returns. While no vault is being opened the refresh comes from
// Fallback.
type CurrentRefresh struct {
	Fallback func() Refresh
	current  atomic.Pointer[Refresh]
}

// Get returns the refresh of the vault being opened, or the fallback while none is.
func (r *CurrentRefresh) Get() Refresh {
	if running := r.current.Load(); running != nil {
		return *running
	}
	return r.Fallback()
}

// Publish makes refresh the refresh of the vault being opened.
func (r *CurrentRefresh) Publish(refresh Refresh) { r.current.Store(&refresh) }

// Retire ends the use of the published refresh.
func (r *CurrentRefresh) Retire() { r.current.Store(nil) }
