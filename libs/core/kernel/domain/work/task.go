// Package work is the unit of work the core shows the user while it runs.
package work

import "context"

// A Task is one unit of work as presented to the user.
type Task struct {
	ID string

	Title string
	About string

	Count, Total int64

	Error string

	IsUserStarted bool

	Kind Kind

	Unit Unit

	PluginID, PluginName string

	Cancel context.CancelFunc
}

// Kind is what an entry of the list of tasks is.
type Kind int

const (
	// KindWork is a piece of work that runs, stops, or fails.
	KindWork Kind = iota
	// KindNotice is a message of the core. It is neither running nor failed.
	KindNotice
)

// Unit is what a piece of work counts. Bytes are read out in the sizes a
// person reads them in, and everything else is counted one by one.
type Unit int

const (
	Things Unit = iota
	Bytes
	// Seconds are a stretch of a recording, read out as a length of time.
	Seconds
)
