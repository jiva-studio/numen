//go:build !linux && !darwin && !windows

package trash

import (
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// moveToTrash is the fallback for platforms with no supported trash.
func moveToTrash(path string, _ time.Time) error {
	return fmt.Errorf("%s: %w", path, machineport.ErrNoTrash)
}
