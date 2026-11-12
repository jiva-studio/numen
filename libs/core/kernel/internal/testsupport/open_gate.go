package testsupport

// OpenGate admits every write, for a test of a use case that has no window to wait for its writes.
type OpenGate struct{}

func (OpenGate) BeginWrite() bool { return true }
func (OpenGate) FinishWrite()     {}
