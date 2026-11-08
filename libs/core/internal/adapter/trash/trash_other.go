//go:build !linux && !darwin && !windows

package trash

import (
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

func moveToTrash(path string, _ time.Time) error {
	return fmt.Errorf("%s: %w", path, machineport.ErrNoTrash)
}
