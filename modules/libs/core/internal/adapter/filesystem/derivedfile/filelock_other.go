//go:build !windows

package derivedfile

// wrapHeldByAnother is err. Here a file is read whatever else has it open.
func wrapHeldByAnother(err error) error { return err }
