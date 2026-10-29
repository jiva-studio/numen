//go:build !windows

package atomicfile

// isFileInUse reports whether a rename failed because another program has the file
// open. Here a file moves over whatever handles are open on it.
func isFileInUse(error) bool { return false }
