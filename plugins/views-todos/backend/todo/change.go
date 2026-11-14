package todo

import "errors"

// Change is what ChangeLine did to a task line.
type Change string

const (
	// ChangeCompleted is an open task that was ticked: its box holds x and it has done: today.
	ChangeCompleted Change = "completed"
	// ChangeRescheduled is an open recurring task that was ticked: its due: moved and it stays open.
	ChangeRescheduled Change = "rescheduled"
	// ChangeReopened is a task that was unticked: its box holds a space and its done: is gone.
	ChangeReopened Change = "reopened"
)

var (
	// ErrNotTask is a line that is not a task line.
	ErrNotTask = errors.New("the line is not a task")
	// ErrNotOpen is a tick of a task that is done or has another status.
	ErrNotOpen = errors.New("the task is not open")
	// ErrNotDone is an untick of a task that is open.
	ErrNotDone = errors.New("the task is not done")
	// ErrInvalidRecurrence is a tick of a task whose rec: cannot be read.
	ErrInvalidRecurrence = errors.New("the recurrence of the task cannot be read")
)

// ChangeLine returns the line after the task is ticked (isDone) or unticked at today, a
// YYYY-MM-DD date, and the change kind. Only the checkbox character and the done: token change;
// every other byte of the line is kept.
func ChangeLine(line string, isDone bool, today string) (string, Change, error) {
	return line, "", errors.New("not implemented")
}
