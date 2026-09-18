//go:build !windows

package filesystem

// asHeld is err. Here a file is read whatever else has it open.
func asHeld(err error) error { return err }
