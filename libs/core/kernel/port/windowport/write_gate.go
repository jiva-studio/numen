package windowport

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	// BeginWrite registers a write and reports false once the window is closing, when nothing may be written.
	BeginWrite() bool
	// FinishWrite ends a write that BeginWrite registered.
	FinishWrite()
}
