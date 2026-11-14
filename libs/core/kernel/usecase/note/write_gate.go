package note

import (
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

func writeUnder[T any](gate windowport.WriteGate, write func() (T, error)) (T, error) {
	// Once the window is closing it fails with window.ErrClosing and does not run write.
	if !gate.BeginWrite() {
		var none T
		return none, windowdomain.ErrClosing
	}
	defer gate.FinishWrite()
	return write()
}
