// Package machinecontract re-exports the machine port contract.
package machinecontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

type (
	Clock        = machineport.Clock
	Delay        = machineport.Delay
	FolderDialog = machineport.FolderDialog
	PathDialog   = machineport.PathDialog
	Trash        = machineport.Trash
	Wait         = machineport.Wait
)

var (
	ErrChoosing       = machine.ErrChoosing
	ErrNoFolderDialog = machine.ErrNoFolderDialog
	ErrNoTrash        = machine.ErrNoTrash
)
