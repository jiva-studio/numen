package container

// openGate admits every write, for an installation with no window that waits for writes.
type openGate struct{}

func (openGate) BeginWrite() bool { return true }
func (openGate) FinishWrite()     {}
