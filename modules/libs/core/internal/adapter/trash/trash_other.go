//go:build !linux && !darwin && !windows

package trash

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
)

// moveToTrash is a platform this application knows no trash on.
func moveToTrash(path string) error {
	return fmt.Errorf("%s: %w", path, machineport.ErrNoTrash)
}
