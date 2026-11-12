package run

import "github.com/jiva-studio/numen/libs/core/kernel/domain/work"

// readingTheSource is the ID of the task that reports a read of the search source that failed.
const readingTheSource = "read search source"

// NewModelReadReporter returns what lists a failed read of the search source as a task that carries the
// failure as its reason, and removes that task when a later read succeeds. Without tasks there is nowhere to list it, and it returns none.
func NewModelReadReporter(tasks *Tasks) func(error) {
	if tasks == nil {
		return nil
	}
	return func(err error) {
		if err == nil {
			tasks.Remove(readingTheSource)
			return
		}
		tasks.Set(work.Task{ID: readingTheSource, Title: "Search source", Error: err.Error()})
	}
}
