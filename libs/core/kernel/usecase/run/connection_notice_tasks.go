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

// Execute adds a notice for the connection set aside and its reason to the list of tasks.
func (u ConnectionNoticeTasks) Execute(setAside connection.SetAside) {
	u.tasks.Set(work.Task{
		ID:    fmt.Sprintf("connection set aside %s %s", setAside.ConnectionID, setAside.Reason),
		Title: "Another connection answered",
		About: getSetAsideSentence(setAside),
		Kind:  work.KindNotice,
	})
}

func getSetAsideSentence(setAside connection.SetAside) string {
	name, next := setAside.ConnectionName, setAside.NextName
	switch setAside.Reason {
	case connection.FailureReasonRejectedKey:
		return fmt.Sprintf("%s rejected the key, so %s answered.", name, next)
	case connection.FailureReasonPaymentRequired:
		return fmt.Sprintf("%s needs payment, so %s answered.", name, next)
	case connection.FailureReasonRateLimited:
		return fmt.Sprintf("%s hit its rate limit, so %s answered.", name, next)
	case connection.FailureReasonInvalidRequest:
		return fmt.Sprintf("%s refused the request, so %s answered.", name, next)
	case connection.FailureReasonUnavailable:
		return fmt.Sprintf("%s is not available, so %s answered.", name, next)
	}
	return fmt.Sprintf("%s is not available, so %s answered.", name, next)
}
