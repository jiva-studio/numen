// Package todos converts the values of the domain to the messages of the service and back.
package todos

import (
	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
)

// EncodeTask returns the message of a task. Its lists are never nil, so they encode as [].
func EncodeTask(task todo.Task) api.Task {
	issues := make([]api.Issue, 0, len(task.Issues))
	for _, issue := range task.Issues {
		issues = append(issues, api.Issue(issue))
	}
	return api.Task{
		Path:       task.Path,
		Line:       task.Line,
		ProseLine:  task.ProseLine,
		Raw:        task.Raw,
		Text:       task.Text,
		Status:     api.Status(task.Status),
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

// EncodeResult returns the message of a change: the refusal and no task when the change was refused.
func EncodeResult(result complete.Result) api.SetTaskDoneResponse {
	var response api.SetTaskDoneResponse
	response.Task = EncodeTask(result.Task)
	if result.Refusal != "" {
		response.Error = api.Refusal(result.Refusal)
		return response
	}
	response.Change = api.Change(result.Change)
	return response
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

// EncodeFields returns the message of the projects, the contexts and the tags. Its lists are never nil.
func EncodeFields(fields index.Fields) api.ListFieldsResponse {
	return api.ListFieldsResponse{Projects: encodeCounts(fields.Projects), Contexts: encodeCounts(fields.Contexts), Tags: encodeCounts(fields.Tags)}
}

func encodeCounts(counts []index.FieldCount) []api.FieldCount {
	encoded := make([]api.FieldCount, 0, len(counts))
	for _, one := range counts {
		encoded = append(encoded, api.FieldCount{Name: one.Name, Count: one.Count})
	}
	return encoded
}
