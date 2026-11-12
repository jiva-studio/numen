package run

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
)

// ConnectionNoticeTasks tells the person, in the list of tasks, that a connection was set aside.
type ConnectionNoticeTasks struct {
	tasks *Tasks
}

// NewConnectionNoticeTasks reports into the list of tasks.
func NewConnectionNoticeTasks(tasks *Tasks) ConnectionNoticeTasks {
	return ConnectionNoticeTasks{tasks: tasks}
}

// Execute adds a notice for the connection and the reason to the list of tasks.
func (u ConnectionNoticeTasks) Execute(connectionID, connectionName string, reason connection.FailureReason) {
	u.tasks.Set(work.Task{
		ID:    fmt.Sprintf("connection set aside %s %s", connectionID, reason),
		Title: "Connection set aside",
		About: fmt.Sprintf("%s was set aside: %s. The next connection in the list answered.", connectionName, reason),
		Kind:  work.KindNotice,
	})
}
