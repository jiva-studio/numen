package note

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func writeUnder[T any](admission vaultport.WriteAdmission, write func() (T, error)) (T, error) {
	release, err := admission.Begin()
	if err != nil {
		var none T
		return none, err
	}
	defer release()
	return write()
}
