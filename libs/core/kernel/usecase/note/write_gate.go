package note

import (
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
)

// WriteGate admits the writes a window waits for before it closes.
type WriteGate interface {
	// BeginWrite registers a write and reports false once the window is closing, when nothing may be written.
	BeginWrite() bool
	// FinishWrite ends a write that BeginWrite registered.
	FinishWrite()
}

// IndexFollow reports whether the index followed a write.
type IndexFollow interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
}

// OpenGate admits every write, for an installation with no window that waits for writes.
type OpenGate struct{}

func (OpenGate) BeginWrite() bool { return true }
func (OpenGate) FinishWrite()     {}

// Once the window is closing it fails with window.ErrClosing and does not run write.
func writeUnder[T any](gate WriteGate, write func() (T, error)) (T, error) {
	if !gate.BeginWrite() {
		var none T
		return none, windowdomain.ErrClosing
	}
	defer gate.FinishWrite()
	return write()
}
