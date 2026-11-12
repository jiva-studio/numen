package run

import "github.com/jiva-studio/numen/libs/core/kernel/domain/work"

// readingTheModel is the ID of the task that reports a read of the embedding model that failed.
const readingTheModel = "read embedding model"

// modelReadFailure is the reason of the task. It holds no text of the failure, which may come from a
// service the person has no part in.
const modelReadFailure = "The embedding model could not be read. Check the embedding settings."

// NewModelReadReporter returns what lists a failed read of the embedding model as a task, and removes that task
// when a later read succeeds. Without tasks there is nowhere to list it, and it returns none.
func NewModelReadReporter(tasks *Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		if err == nil {
			tasks.Remove(readingTheModel)
			return
		}
		tasks.Set(work.Task{ID: readingTheModel, Title: "Read embedding model", Error: modelReadFailure})
	}
}
