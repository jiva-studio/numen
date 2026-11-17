//go:build !windows

package atomicfile

func isFileInUse(error) bool { return false }
