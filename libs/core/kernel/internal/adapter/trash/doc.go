// Package trash moves a folder to the location where this machine keeps deleted
// items, from which the user can restore it.
//
// Each platform has its own implementation: the freedesktop directories on
// Linux, NSFileManager on macOS, the recycle bin on Windows.
package trash
