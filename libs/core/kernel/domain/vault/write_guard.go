package vault

import "errors"

// CheckWrite allows a write to a vault that nobody manages, or by the plugin that manages it. Any other caller,
// the window included (caller empty), gets ErrReadOnly.
func CheckWrite(_, _ string) error {
	return errors.New("not implemented")
}
