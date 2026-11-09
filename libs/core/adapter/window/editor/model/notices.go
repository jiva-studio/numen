package model

import (
	"fmt"

	modelcallport "github.com/jiva-studio/numen/libs/core/usecase/modelcall/modelcallcontract"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// Notices tells the person, in the list of tasks, that a connection was set aside.
type Notices struct {
	tasks *run.Tasks
}

// NewNotices reports into the list of tasks.
func NewNotices(tasks *run.Tasks) Notices { return Notices{tasks: tasks} }

// ReportSetAside adds a notice for the connection and the reason to the list of tasks.
func (n Notices) ReportSetAside(connectionID, connectionName string, reason modelcallport.FailureReason) {
	n.tasks.Set(run.Task{
		ID:    fmt.Sprintf("connection set aside %s %s", connectionID, reason),
		Title: "Connection set aside",
		About: fmt.Sprintf("%s was set aside: %s. The next connection in the list answered.", connectionName, reason),
		Kind:  run.KindNotice,
	})
}
