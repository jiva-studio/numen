//go:build windows

package vaultreader

import (
	"os"
	"syscall"
)

// openReadable opens a file for reading.
func openReadable(path string) (*os.File, error) {
	return os.Open(path)
}

// openShared opens a file for reading with read, write and delete sharing, so
// the file can be renamed or deleted while it is open.
func openShared(path string) (*os.File, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	const share = syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, share, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(handle), path), nil
}
