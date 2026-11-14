package filter

import (
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

const today = "2026-11-14"

func open(path string, line int, text, due string) todo.Task {
	return todo.Task{Path: path, Line: line, Raw: "- [ ] " + text, Text: text, Status: todo.StatusOpen, Due: due}
}

func getTexts(tasks []todo.Task) []string {
	out := []string{}
	for _, task := range tasks {
		out = append(out, task.Text)
	}
	return out
}

func apply(tasks []todo.Task, f Filter) []string {
	return getTexts(Apply(tasks, f, today))
}

func TestAnOpenTaskDueBeforeTodayIsOverdueAndOneDueTodayIsToday(t *testing.T) {
	tasks := []todo.Task{
		open("a.md", 1, "yesterday", "2026-11-13"),
		open("a.md", 2, "today", "2026-11-14"),
		open("a.md", 3, "tomorrow", "2026-11-15"),
		open("a.md", 4, "no date", ""),
	}

	if got := apply(tasks, Filter{Due: DueOverdue}); !reflect.DeepEqual(got, []string{"yesterday"}) {
		t.Errorf("overdue = %v", got)
	}
	if got := apply(tasks, Filter{Due: DueToday}); !reflect.DeepEqual(got, []string{"today"}) {
		t.Errorf("today = %v", got)
	}
}

func TestAnOpenTaskWithNoDateIsNoDateAndAFinishedTaskIsNotCounted(t *testing.T) {
	done := open("a.md", 3, "finished", "")
	done.Status = todo.StatusDone
	dated := open("a.md", 4, "dated", "2026-11-20")
	invalid := open("a.md", 5, "bad date", "")
	invalid.Issues = []todo.Issue{todo.IssueInvalidDue}
	tasks := []todo.Task{open("a.md", 1, "plain", ""), open("a.md", 2, "plain too", ""), done, dated, invalid}

	if got := apply(tasks, Filter{Due: DueNone}); !reflect.DeepEqual(got, []string{"plain", "plain too", "bad date"}) {
		t.Errorf("none = %v", got)
	}
}

func TestAFinishedTaskDueInThePastIsNotOverdue(t *testing.T) {
	done := open("a.md", 1, "finished", "2026-11-01")
	done.Status = todo.StatusDone

	if got := apply([]todo.Task{done}, Filter{Due: DueOverdue}); len(got) != 0 {
		t.Errorf("overdue = %v, want none", got)
	}
}

func TestTheStatusFilterShowsOpenTasksByDefaultAndOtherStatusesOnRequest(t *testing.T) {
	done := open("a.md", 2, "done one", "")
	done.Status = todo.StatusDone
	other := open("a.md", 3, "cancelled one", "")
	other.Status = todo.StatusOther
	tasks := []todo.Task{open("a.md", 1, "open one", ""), done, other}

	cases := []struct {
		name   string
		status []todo.Status
		want   []string
	}{
		{"an empty list is open", nil, []string{"open one"}},
		{"done", []todo.Status{todo.StatusDone}, []string{"done one"}},
		{"open and other", []todo.Status{todo.StatusOpen, todo.StatusOther}, []string{"open one", "cancelled one"}},
		{"all three", []todo.Status{todo.StatusOpen, todo.StatusDone, todo.StatusOther}, []string{"open one", "done one", "cancelled one"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := apply(tasks, Filter{Status: tc.status}); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestADueFilterOfAnyOrNoneSetAcceptsEveryOpenTask(t *testing.T) {
	tasks := []todo.Task{open("a.md", 1, "dated", "2026-12-01"), open("a.md", 2, "plain", "")}

	for _, due := range []DueFilter{"", DueAny} {
		if got := apply(tasks, Filter{Due: due}); !reflect.DeepEqual(got, []string{"dated", "plain"}) {
			t.Errorf("due %q = %v", due, got)
		}
	}
}

func TestTheRangeIncludesItsFirstAndLastDay(t *testing.T) {
	tasks := []todo.Task{
		open("a.md", 1, "before", "2026-11-09"),
		open("a.md", 2, "first", "2026-11-10"),
		open("a.md", 3, "middle", "2026-11-15"),
		open("a.md", 4, "last", "2026-11-20"),
		open("a.md", 5, "after", "2026-11-21"),
		open("a.md", 6, "no date", ""),
	}

	got := apply(tasks, Filter{Due: DueRange, From: "2026-11-10", To: "2026-11-20"})

	if want := []string{"first", "middle", "last"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFiltersCombineWithAnd(t *testing.T) {
	work := open("a.md", 1, "report", "2026-11-10")
	work.Projects, work.Contexts, work.Priority = []string{"work"}, []string{"desk"}, "A"
	home := open("a.md", 2, "dishes", "2026-11-10")
	home.Projects, home.Contexts, home.Priority = []string{"home"}, []string{"desk"}, "B"
	both := open("b.md", 1, "Report budget", "2026-11-10")
	both.Projects, both.Contexts, both.Priority = []string{"work", "home"}, []string{"phone"}, "A"
	tasks := []todo.Task{work, home, both}

	cases := []struct {
		name string
		f    Filter
		want []string
	}{
		{"a project, any of the list", Filter{Projects: []string{"home", "none"}}, []string{"Report budget", "dishes"}},
		{"a context", Filter{Contexts: []string{"desk"}}, []string{"report", "dishes"}},
		{"a priority", Filter{Priorities: []string{"A"}}, []string{"report", "Report budget"}},
		{"a text in any case", Filter{Text: "REPORT"}, []string{"report", "Report budget"}},
		{"a project and a context", Filter{Projects: []string{"work"}, Contexts: []string{"desk"}}, []string{"report"}},
		{"a project, a priority and a text", Filter{Projects: []string{"work"}, Priorities: []string{"A"}, Text: "budget"}, []string{"Report budget"}},
		{"a note path", Filter{Path: "b.md"}, []string{"Report budget"}},
		{"a due filter and a project", Filter{Due: DueOverdue, Projects: []string{"home"}}, []string{"Report budget", "dishes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := apply(tasks, tc.f); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAPathEndingInASlashIsAFolderPrefixAndOtherPathsNameOneNote(t *testing.T) {
	tasks := []todo.Task{
		open("projects/a.md", 1, "in projects", ""),
		open("projects/deep/b.md", 1, "deep", ""),
		open("projects.md", 1, "a note named projects", ""),
		open("other/c.md", 1, "elsewhere", ""),
	}

	if got := apply(tasks, Filter{Path: "projects/"}); !reflect.DeepEqual(got, []string{"in projects", "deep"}) {
		t.Errorf("folder = %v", got)
	}
	if got := apply(tasks, Filter{Path: "projects.md"}); !reflect.DeepEqual(got, []string{"a note named projects"}) {
		t.Errorf("note = %v", got)
	}
}

func TestTheResultIsOrderedByDueThenPriorityThenPathThenLine(t *testing.T) {
	task := func(path string, line int, text, due, priority string) todo.Task {
		made := open(path, line, text, due)
		made.Priority = priority
		return made
	}
	tasks := []todo.Task{
		task("z.md", 1, "no date, no priority", "", ""),
		task("b.md", 2, "later", "2026-12-01", "A"),
		task("b.md", 9, "same day, no priority, line 9", "2026-11-10", ""),
		task("b.md", 1, "same day, no priority, line 1", "2026-11-10", ""),
		task("a.md", 5, "same day, B", "2026-11-10", "B"),
		task("c.md", 1, "same day, A, c", "2026-11-10", "A"),
		task("a.md", 3, "same day, A, a", "2026-11-10", "A"),
		task("a.md", 1, "no date, A", "", "A"),
	}

	want := []string{
		"same day, A, a",
		"same day, A, c",
		"same day, B",
		"same day, no priority, line 1",
		"same day, no priority, line 9",
		"later",
		"no date, A",
		"no date, no priority",
	}
	if got := apply(tasks, Filter{}); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestApplyKeepsTheOrderOfItsInput(t *testing.T) {
	tasks := []todo.Task{open("b.md", 1, "second", "2026-12-01"), open("a.md", 1, "first", "2026-11-01")}

	Apply(tasks, Filter{}, today)

	if got := getTexts(tasks); !reflect.DeepEqual(got, []string{"second", "first"}) {
		t.Errorf("the input was reordered: %v", got)
	}
}
