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
}

// NewProgress returns a progress with no restore, no link and no revoke.
func NewProgress() *Progress {
	store := newPendingStore()
	return &Progress{restores: store, pending: store, links: newLinkStore(), gate: NewCreateGate()}
}

// GetCreateGate returns the gate that serializes the check for an absent identity and the write of one over every use case of this progress.
func (p *Progress) GetCreateGate() CreateGate { return p.gate }
