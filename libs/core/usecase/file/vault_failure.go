package file

import "errors"

func newVaultFailure(err error) error {
	return errors.New(err.Error())
}
