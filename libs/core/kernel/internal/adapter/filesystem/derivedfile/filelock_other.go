//go:build !windows

package derivedfile

func wrapLockedByAnother(err error) error { return err }
