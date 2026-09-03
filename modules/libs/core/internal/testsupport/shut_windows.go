package testsupport

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Shut makes a file nobody may open, and opens it again when the test ends.
// Here that is another program holding the file with no share at all, which is
// how a file arrives closed on Windows.
func Shut(tb testing.TB, path string) {
	tb.Helper()
	tb.Cleanup(closing(tb, hold(tb, path, 0)))
}

// Unwritable makes a file nothing may save over, and lets it be saved again
// when the test ends. The file is held open for reading and shared for reading
// alone: it reads as it always did, and a save renaming over the name is
// refused for as long as the handle stands.
func Unwritable(tb testing.TB, path string) {
	tb.Helper()
	tb.Cleanup(closing(tb, hold(tb, path, windows.FILE_SHARE_READ)))
}

// hold opens a file for reading under the share given and keeps it open.
func hold(tb testing.TB, path string, share uint32) windows.Handle {
	tb.Helper()
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		tb.Fatal(err)
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		share,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		tb.Fatal(err)
	}
	return handle
}

func closing(tb testing.TB, handle windows.Handle) func() {
	return func() {
		if err := windows.CloseHandle(handle); err != nil {
			tb.Error(err)
		}
	}
}
