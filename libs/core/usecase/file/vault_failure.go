package file

import "errors"

// newVaultFailure is an error with the text of err that wraps nothing, so no code chosen by what an error wraps matches it.
func newVaultFailure(err error) error {
	return errors.New(err.Error())
}
