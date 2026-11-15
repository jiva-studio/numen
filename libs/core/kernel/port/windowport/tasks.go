package windowport

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

// Tasks is the list of what the window is doing.
type Tasks interface {
	AddTask(task work.Task)
	RemoveTask(id string)
}
