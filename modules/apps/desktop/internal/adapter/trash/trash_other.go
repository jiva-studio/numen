//go:build !linux && !darwin && !windows

package trash

import (
	"fmt"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// send is a platform this application knows no trash on.
func send(path string) error {
	return fmt.Errorf("%s: %w", path, port.ErrNoTrash)
}
