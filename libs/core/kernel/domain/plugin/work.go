package plugin

// Task is one piece of a plugin's work, as shown to the user. ID is the
// plugin's own identifier, and Done and Total are counted in Unit.
type Task struct {
	ID            string
	Title         string
	About         string
	Done          int64
	Total         int64
	Unit          TaskUnit
	Error         string
	IsUserStarted bool
}

// TaskUnit is what a task counts.
type TaskUnit int

const (
	TaskUnitThings TaskUnit = iota
	TaskUnitBytes
	TaskUnitSeconds
)
