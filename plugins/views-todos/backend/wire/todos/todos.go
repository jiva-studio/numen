// Package todos converts the values of the domain to the messages of the service and back.
package todos

import (
	"github.com/jiva-studio/numen/plugins/views-todos/backend/api"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

// EncodeTask returns the message of a task.
func EncodeTask(_ todo.Task) api.Task {
	return api.Task{}
}

// DecodeFilter returns the filter a message asks for.
func DecodeFilter(_ api.Filter) filter.Filter {
	return filter.Filter{}
}
