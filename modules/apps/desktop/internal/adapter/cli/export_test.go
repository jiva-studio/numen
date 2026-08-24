package cli

import (
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ErasesInto puts a trash of a test's own where an erased folder goes, and
// answers with what puts the machine's back. A test sends nothing to the trash
// of the machine it runs on.
func ErasesInto(trash port.Trash) func() {
	was := erasesInto
	erasesInto = func(container.Config) port.Trash { return trash }
	return func() { erasesInto = was }
}
