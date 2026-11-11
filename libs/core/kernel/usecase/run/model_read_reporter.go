package run

import "github.com/jiva-studio/numen/libs/core/kernel/domain/work"

// readingTheModel is the ID of the task that reports a read of the embedding model that failed.
const readingTheModel = "read embedding model"

// NewModelReadReporter returns what lists a failed read of the embedding model as a task that carries the
// failure as its reason. Without tasks there is nowhere to list it, and it returns none.
func NewModelReadReporter(tasks *Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		tasks.Set(work.Task{ID: readingTheModel, Title: "Read embedding model", Error: err.Error()})
	}
}
