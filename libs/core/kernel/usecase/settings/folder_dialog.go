package settings

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// PathDialog lets the user pick a file or a folder on this machine.
type PathDialog interface {
	machineport.FolderDialog
	// ChooseFile shows the machine's native file picker, listing files whose
	// extension is one of accept, or every file when accept is empty. It behaves like ChooseFolder.
	ChooseFile(ctx context.Context, title, startingAt string, accept []string) (string, bool, error)
}
