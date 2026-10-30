// Package machinecontract re-exports the machine port contract.
package machinecontract

import "github.com/jiva-studio/numen/libs/core/port/machineport"

type (
	Clock        = machineport.Clock
	Delay        = machineport.Delay
	FolderDialog = machineport.FolderDialog
	HomeFolder   = machineport.HomeFolder
	PathDialog   = machineport.PathDialog
	Trash        = machineport.Trash
	Wait         = machineport.Wait
)

var (
	ErrChoosing       = machineport.ErrChoosing
	ErrNoFolderDialog = machineport.ErrNoFolderDialog
	ErrNoTrash        = machineport.ErrNoTrash
)
