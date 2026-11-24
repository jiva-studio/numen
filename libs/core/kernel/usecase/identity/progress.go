package identity

// Progress keeps in memory what the identity use cases have in progress: the restore that waits for the chain of a
// sync plugin, and the link and the revoke that wait for the other device and the server. The use cases of the
// windows and of the plugins of one installation share one.
type Progress struct {
	restores restoreHold
	pending  restoreReader
	links    linkHold
	created  generatedWords
	gate     CreateGate
	changes  *changeSignal
}

// NewProgress returns a progress with no restore, no link and no revoke.
func NewProgress() *Progress {
	signal := &changeSignal{}
	store := newPendingStore(signal)
	return &Progress{restores: store, pending: store, links: newLinkStore(signal), gate: NewCreateGate(), changes: signal}
}

// GetCreateGate returns the gate that serializes the check for an absent identity and the write of one over every use case of this progress.
func (p *Progress) GetCreateGate() CreateGate { return p.gate }

// OnChange sets a function that is called when the record of the identity is written or deleted, or the restore or the
// link in progress changes. It is called while a lock is held, so it returns at once and calls no identity use case.
func (p *Progress) OnChange(listener func()) { p.changes.subscribe(listener) }
