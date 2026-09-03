//go:build windows

package trash

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// What the shell is asked for: a delete the recycle bin can undo, with no
// window, no question and no error of its own to show.
const (
	deletion       = 0x0003
	allowUndo      = 0x0040
	noConfirmation = 0x0010
	silent         = 0x0004
	noErrorUI      = 0x0400
)

// shFileOpStruct is SHFILEOPSTRUCTW.
type shFileOpStruct struct {
	window        windows.Handle
	function      uint32
	from          *uint16
	to            *uint16
	flags         uint16
	aborted       int32
	nameMappings  uintptr
	progressTitle *uint16
}

var shFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// send puts the folder in the recycle bin.
func send(path string) error {
	// The shell reads a list of paths, and the list ends with a NUL of its own
	// after the one that ends the last path in it.
	from, err := windows.UTF16FromString(path)
	if err != nil {
		return err
	}
	from = append(from, 0)

	op := shFileOpStruct{
		function: deletion,
		from:     &from[0],
		flags:    allowUndo | noConfirmation | noErrorUI | silent,
	}
	if rc, _, _ := shFileOperationW.Call(uintptr(unsafe.Pointer(&op))); rc != 0 {
		return fmt.Errorf("%s: the shell would not delete it (0x%x)", path, rc)
	}
	if op.aborted != 0 {
		return fmt.Errorf("%s: the deletion stopped part way", path)
	}
	return nil
}
