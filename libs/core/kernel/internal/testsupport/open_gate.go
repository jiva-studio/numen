package testsupport

// OpenGate admits every write, for a use case under test with no window that waits for writes.
type OpenGate struct{}

func (OpenGate) BeginWrite() bool { return true }
func (OpenGate) FinishWrite()     {}
