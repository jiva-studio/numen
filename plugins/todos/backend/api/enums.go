package api

// Status is what the character of a checkbox says.
type Status string

// Status values.
const (
	StatusUnspecified Status = ""
	StatusOpen        Status = "open"
	StatusDone        Status = "done"
	StatusOther       Status = "other"
)

// Issue is a field of a task line that is written but cannot be read.
type Issue string

// Issue values.
const (
	IssueUnspecified       Issue = ""
	IssueInvalidDue        Issue = "invalidDue"
	IssueInvalidDone       Issue = "invalidDone"
	IssueInvalidRecurrence Issue = "invalidRecurrence"
	IssueInvalidPriority   Issue = "invalidPriority"
)

// DueFilter is which due dates a filter accepts.
type DueFilter string

// DueFilter values.
const (
	DueFilterUnspecified   DueFilter = ""
	DueFilterAny           DueFilter = "any"
	DueFilterNone          DueFilter = "none"
	DueFilterOverdue       DueFilter = "overdue"
	DueFilterToday         DueFilter = "today"
	DueFilterThisWeek      DueFilter = "thisWeek"
	DueFilterNextSevenDays DueFilter = "nextSevenDays"
	DueFilterRange         DueFilter = "range"
)

// Change is what a tick or an untick did to the line.
type Change string

// Change values.
const (
	ChangeUnspecified Change = ""
	ChangeCompleted   Change = "completed"
	ChangeRescheduled Change = "rescheduled"
	ChangeReopened    Change = "reopened"
)

// Refusal is why a change was not written.
type Refusal string

// Refusal values.
const (
	RefusalUnspecified       Refusal = ""
	RefusalNotFound          Refusal = "notFound"
	RefusalUnreadable        Refusal = "unreadable"
	RefusalStale             Refusal = "stale"
	RefusalInvalidRecurrence Refusal = "invalidRecurrence"
	RefusalNotOpen           Refusal = "notOpen"
	RefusalNotDone           Refusal = "notDone"
)
