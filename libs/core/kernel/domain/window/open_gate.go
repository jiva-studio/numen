package window

// OpenGate admits every write, for an installation with no window that waits for writes.
type OpenGate struct{}

func (OpenGate) BeginWrite() bool { return true }
func (OpenGate) FinishWrite()     {}
