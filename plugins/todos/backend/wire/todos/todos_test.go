package todos

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	"github.com/jiva-studio/numen/plugins/todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

func TestEncodeTaskCopiesEveryFieldAndEncodesEmptyListsAsArrays(t *testing.T) {
	got := EncodeTask(todo.Task{
		Path: "Projects/Home.md", Line: 7, Raw: "- [ ] (A) buy milk due:2026-11-13 +home @phone", Text: "buy milk",
		Status: todo.StatusOpen, Priority: "A", Due: "2026-11-13", DoneOn: "2026-11-01", Recurrence: "rec:1w",
		Projects: []string{"home"}, Contexts: []string{"phone"}, Issues: []todo.Issue{todo.IssueInvalidDone}, Depth: 2,
	})

	want := api.Task{
		Path: "Projects/Home.md", Line: 7, Raw: "- [ ] (A) buy milk due:2026-11-13 +home @phone", Text: "buy milk",
		Status: "open", Priority: "A", Due: "2026-11-13", DoneOn: "2026-11-01", Recurrence: "rec:1w",
		Projects: []string{"home"}, Contexts: []string{"phone"}, Issues: []string{"invalidDone"}, Depth: 2,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("task = %+v, want %+v", got, want)
	}

	plain := EncodeTask(todo.Task{Path: "a.md", Line: 1, Raw: "- [ ] x", Text: "x", Status: todo.StatusDone})
	if plain.Projects == nil || plain.Contexts == nil || plain.Issues == nil {
		t.Errorf("lists = %v %v %v, want empty lists that encode as []", plain.Projects, plain.Contexts, plain.Issues)
	}
}

func TestDecodeFilterCopiesEveryField(t *testing.T) {
	got := DecodeFilter(api.Filter{
		Status: []string{"open", "done"}, Due: "range", From: "2026-11-01", To: "2026-11-30", Projects: []string{"home"},
		Contexts: []string{"phone"}, Priorities: []string{"A"}, Path: "Projects/", Text: "milk",
	})

	want := filter.Filter{
		Status: []todo.Status{todo.StatusOpen, todo.StatusDone}, Due: filter.DueRange, From: "2026-11-01", To: "2026-11-30",
		Projects: []string{"home"}, Contexts: []string{"phone"}, Priorities: []string{"A"}, Path: "Projects/", Text: "milk",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filter = %+v, want %+v", got, want)
	}
}
