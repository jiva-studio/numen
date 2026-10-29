//go:build !windows

package derivedfile

// wrapLockedByAnother is err. Here a file is read whatever else has it open.
func wrapLockedByAnother(err error) error { return err }
