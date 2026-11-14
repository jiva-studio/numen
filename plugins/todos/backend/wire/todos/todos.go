// Package todos converts the values of the domain to the messages of the service and back.
package todos

import (
	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

// EncodeTask returns the message of a task. Its lists are never nil, so they encode as [].
func EncodeTask(task todo.Task) api.Task {
	issues := make([]string, 0, len(task.Issues))
	for _, issue := range task.Issues {
		issues = append(issues, string(issue))
	}
	return api.Task{
		Path:       task.Path,
		Line:       task.Line,
		Raw:        task.Raw,
		Text:       task.Text,
		Status:     string(task.Status),
		Priority:   task.Priority,
		Due:        task.Due,
		DoneOn:     task.DoneOn,
		Recurrence: task.Recurrence,
		Projects:   append([]string{}, task.Projects...),
		Contexts:   append([]string{}, task.Contexts...),
		Issues:     issues,
		Depth:      task.Depth,
	}
}

// DecodeFilter returns the filter a message asks for.
func DecodeFilter(message api.Filter) filter.Filter {
	statuses := make([]todo.Status, 0, len(message.Status))
	for _, status := range message.Status {
		statuses = append(statuses, todo.Status(status))
	}
	return filter.Filter{
		Status:     statuses,
		Due:        filter.DueFilter(message.Due),
		From:       message.From,
		To:         message.To,
		Projects:   message.Projects,
		Contexts:   message.Contexts,
		Priorities: message.Priorities,
		Path:       message.Path,
		Text:       message.Text,
	}
}
