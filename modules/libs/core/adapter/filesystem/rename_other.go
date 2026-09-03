//go:build !windows

package filesystem

// transient says whether a rename failed because another program holds the file
// open. Here a file moves over whatever handles are open on it.
func transient(error) bool { return false }
