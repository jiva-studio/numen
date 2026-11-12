//go:build !linux && !darwin && !windows

package trash

import (
	"fmt"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"time"
)

func moveToTrash(path string, _ time.Time) error {
	return fmt.Errorf("%s: %w", path, machine.ErrNoTrash)
}
