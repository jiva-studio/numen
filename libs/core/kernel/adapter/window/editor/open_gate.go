package editor

// OpenGate admits every write, for an installation with no window that waits for writes.
type OpenGate struct{}

// BeginWrite admits the write.
func (OpenGate) BeginWrite() bool { return true }

// FinishWrite does nothing: no window waits for the write.
func (OpenGate) FinishWrite() {}
