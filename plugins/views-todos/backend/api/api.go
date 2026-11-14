// Package api declares the JSON messages of the todos service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.views.todos"

const (
	MethodGetSummary   = "GetSummary"
	MethodListTasks    = "ListTasks"
	MethodRefreshIndex = "RefreshIndex"
	MethodWatchTasks   = "WatchTasks"
)

// Task is one task line of a note.
type Task struct {
	Path       string   `json:"path"`
	Line       int      `json:"line"`
	Raw        string   `json:"raw"`
	Text       string   `json:"text"`
	Status     string   `json:"status"`
	Priority   string   `json:"priority"`
	Due        string   `json:"due"`
	DoneOn     string   `json:"doneOn"`
	Recurrence string   `json:"recurrence"`
	Projects   []string `json:"projects"`
	Contexts   []string `json:"contexts"`
	Issues     []string `json:"issues"`
	Depth      int      `json:"depth"`
}

// Filter is what ListTasks selects by; its fields combine with AND.
type Filter struct {
	Status     []string `json:"status"`
	Due        string   `json:"due"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Projects   []string `json:"projects"`
	Contexts   []string `json:"contexts"`
	Priorities []string `json:"priorities"`
	Path       string   `json:"path"`
	Text       string   `json:"text"`
}

// GetSummaryRequest asks for the counts at a day.
type GetSummaryRequest struct {
	Today string `json:"today"`
}

// Summary counts the open tasks.
type Summary struct {
	IsReady         bool `json:"isReady"`
	OverdueCount    int  `json:"overdueCount"`
	TodayCount      int  `json:"todayCount"`
	NoDateCount     int  `json:"noDateCount"`
	UnreadableCount int  `json:"unreadableCount"`
}

// ListTasksRequest asks for the tasks a filter accepts. A limit of zero is 500; the most is 2000.
type ListTasksRequest struct {
	Today  string `json:"today"`
	Filter Filter `json:"filter"`
	Limit  int    `json:"limit"`
}

// ListTasksResponse is the tasks of a request. Total is the count before the limit.
type ListTasksResponse struct {
	Tasks    []Task `json:"tasks"`
	Total    int    `json:"total"`
	Revision uint64 `json:"revision"`
}

// Rename is a note that moved.
type Rename struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// RefreshIndexRequest names the notes that changed. No path and no rename is a full check.
type RefreshIndexRequest struct {
	Paths   []string `json:"paths"`
	Renamed []Rename `json:"renamed"`
}

// RefreshIndexResponse is empty.
type RefreshIndexResponse struct{}

// WatchTasksRequest opens the stream of changes.
type WatchTasksRequest struct{}

// WatchTasksMessage is sent once at the start and after each change of the index.
type WatchTasksMessage struct {
	Revision uint64 `json:"revision"`
	IsReady  bool   `json:"isReady"`
}
