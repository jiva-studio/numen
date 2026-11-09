package pluginport

import "context"

// PluginWork is the core's shared queue of heavy work and its list of tasks, as
// the plugins use them. Every call names the plugin it is made for.
type PluginWork interface {
	// EnqueueWork waits for the plugin's turn in the queue and returns it.
	// onWaiting is called where other work holds the queue. A context that
	// ends while waiting takes no turn.
	EnqueueWork(ctx context.Context, pluginID string, isUserStarted bool, onWaiting func()) (Lease, error)
	// ReportTask puts one of the plugin's tasks in the list, or replaces it.
	ReportTask(pluginID string, task PluginTask)
	// FinishTask takes one of the plugin's tasks out of the list.
	FinishTask(pluginID, taskID string)
	// ClearPlugin takes every task of the plugin out of the list and ends its
	// turns and its waits.
	ClearPlugin(pluginID string)
}

// Lease is a plugin's hold on the queue.
type Lease interface {
	// Release ends the lease and lets the next work in the queue run. A lease
	// released twice is released once.
	Release()
	// WatchWaiting returns a channel closed once other work waits for the queue.
	WatchWaiting() <-chan struct{}
	// WatchEnd returns a channel closed once the lease is over.
	WatchEnd() <-chan struct{}
}

// PluginTask is one piece of a plugin's work, as shown to the user. ID is the
// plugin's own identifier, and Done and Total are counted in Unit.
type PluginTask struct {
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
