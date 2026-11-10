// Package machinecontract re-exports the machine port contract.
package machinecontract

import "github.com/jiva-studio/numen/libs/core/kernel/port/machineport"

type (
	Clock        = machineport.Clock
	FolderDialog = machineport.FolderDialog
	PathDialog   = machineport.PathDialog
	Trash        = machineport.Trash
)
