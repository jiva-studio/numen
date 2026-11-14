package todo

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// shown is the part of a task a case states; the other fields are checked by the cases about them.
type shown struct {
	Line   int
	Status Status
	Raw    string
	Text   string
}

func show(tasks []Task) []shown {
	out := make([]shown, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, shown{Line: task.Line, Status: task.Status, Raw: task.Raw, Text: task.Text})
	}
	return out
}

func lines(text ...string) []byte { return []byte(strings.Join(text, "\n")) }

func parse(t *testing.T, raw []byte) []Task {
	t.Helper()
	tasks, err := ParseNote(raw)
	if err != nil {
		t.Fatalf("ParseNote: %v", err)
	}
	return tasks
}

func TestParseNoteReadsWhichLinesAreTasks(t *testing.T) {
	cases := []struct {
		name string
		note []byte
		want []shown
	}{
		{
			name: "a line with a dash and an empty box is an open task",
			note: lines("- [ ] buy milk"),
			want: []shown{{1, StatusOpen, "- [ ] buy milk", "buy milk"}},
		},
		{
			name: "star, plus, dot and parenthesis markers make tasks",
			note: lines("* [ ] a", "+ [ ] b", "1. [ ] c", "20) [ ] d"),
			want: []shown{
				{1, StatusOpen, "* [ ] a", "a"},
				{2, StatusOpen, "+ [ ] b", "b"},
				{3, StatusOpen, "1. [ ] c", "c"},
				{4, StatusOpen, "20) [ ] d", "d"},
			},
		},
		{
			name: "x and X are done and any other character is another status",
			note: lines("- [x] a", "- [X] b", "- [-] c", "- [/] d", "- [ ] e"),
			want: []shown{
				{1, StatusDone, "- [x] a", "a"},
				{2, StatusDone, "- [X] b", "b"},
				{3, StatusOther, "- [-] c", "c"},
				{4, StatusOther, "- [/] d", "d"},
				{5, StatusOpen, "- [ ] e", "e"},
			},
		},
		{
			name: "a box with no marker, no space, no character, two characters or no text is not a task",
			note: lines("[ ] a", "-[ ] b", "- [] c", "- [ab] d", "- [ ]", "- [ ]   ", "- [ ]e", "word - [ ] f", "1.[ ] g"),
			want: []shown{},
		},
		{
			name: "a line in a backtick fence or a tilde fence is not a task",
			note: lines("```", "- [ ] hidden", "```", "~~~", "- [ ] hidden too", "~~~", "- [ ] shown"),
			want: []shown{{7, StatusOpen, "- [ ] shown", "shown"}},
		},
		{
			name: "a line in an HTML comment is not a task, on one line or over several",
			note: lines("<!--", "- [ ] hidden", "-->", "<!-- - [ ] hidden too -->", "- [ ] shown"),
			want: []shown{{5, StatusOpen, "- [ ] shown", "shown"}},
		},
		{
			name: "a line in the frontmatter is not a task and the line numbers count it",
			note: lines("---", "summary: |", "  - [ ] hidden", "---", "- [ ] shown"),
			want: []shown{{5, StatusOpen, "- [ ] shown", "shown"}},
		},
		{
			name: "a task in a quote or a callout is found with its prefix in the raw line",
			note: lines("> - [ ] quoted", "> [!todo]", "> - [ ] in a callout", ">> - [ ] twice quoted"),
			want: []shown{
				{1, StatusOpen, "> - [ ] quoted", "quoted"},
				{3, StatusOpen, "> - [ ] in a callout", "in a callout"},
				{4, StatusOpen, ">> - [ ] twice quoted", "twice quoted"},
			},
		},
		{
			name: "a text with no list open and four spaces of indentation is a code block",
			note: lines("a paragraph", "", "    - [ ] code"),
			want: []shown{},
		},
		{
			name: "a line indented by four spaces below an open list item is a task",
			note: lines("- an item", "    - [ ] sub"),
			want: []shown{{2, StatusOpen, "    - [ ] sub", "sub"}},
		},
		{
			name: "a line without a final line break is a task",
			note: []byte("- [ ] last"),
			want: []shown{{1, StatusOpen, "- [ ] last", "last"}},
		},
		{
			name: "a line is not changed when it holds unknown words and pairs",
			note: lines("- [ ] x #tag https://a.example/c ^abc owner:me"),
			want: []shown{{1, StatusOpen, "- [ ] x #tag https://a.example/c ^abc owner:me", "x #tag https://a.example/c ^abc owner:me"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := show(parse(t, tc.note))
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("tasks = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseNoteFindsATaskAtAnyDepth(t *testing.T) {
	tasks := parse(t, lines("- [ ] top", "  - [ ] child", "    - [ ] grandchild", "      - [ ] great", "\t- [ ] tabbed"))

	if len(tasks) != 5 {
		t.Fatalf("got %d tasks, want 5", len(tasks))
	}
	for line, depth := range map[int]int{1: 0, 2: 1, 3: 2, 4: 3} {
		if tasks[line-1].Depth != depth {
			t.Errorf("line %d depth = %d, want %d", line, tasks[line-1].Depth, depth)
		}
	}
}

func TestParseNoteCountsLinesOfTheFileWhateverTheLineEnding(t *testing.T) {
	text := []string{"---", "title: Plan", "---", "", "- [ ] first", "- [x] second"}
	unix := parse(t, []byte(strings.Join(text, "\n")))
	windows := parse(t, []byte(strings.Join(text, "\r\n")))

	if got := show(unix); !reflect.DeepEqual(got, []shown{{5, StatusOpen, "- [ ] first", "first"}, {6, StatusDone, "- [x] second", "second"}}) {
		t.Errorf("LF note: tasks = %+v", got)
	}
	if !reflect.DeepEqual(show(unix), show(windows)) {
		t.Errorf("CRLF note: tasks = %+v, want the tasks of the LF note %+v", show(windows), show(unix))
	}
}

func TestParseNoteCountsLinesAfterAByteOrderMark(t *testing.T) {
	tasks := parse(t, []byte("\xef\xbb\xbf---\ntitle: x\n---\n- [ ] a"))

	if got := show(tasks); !reflect.DeepEqual(got, []shown{{4, StatusOpen, "- [ ] a", "a"}}) {
		t.Errorf("tasks = %+v", got)
	}
}

func TestParseNoteRefusesANoteWhoseFrontmatterIsNeverClosed(t *testing.T) {
	tasks, err := ParseNote(lines("---", "title: x", "- [ ] a"))

	if !errors.Is(err, ErrUnreadable) {
		t.Errorf("error = %v, want ErrUnreadable", err)
	}
	if len(tasks) != 0 {
		t.Errorf("tasks = %+v, want none", tasks)
	}
}

func TestParseNoteRefusesANoteWhoseFrontmatterIsNotAMapping(t *testing.T) {
	_, err := ParseNote(lines("---", "title: [unclosed", "---", "- [ ] a"))

	if !errors.Is(err, ErrUnreadable) {
		t.Errorf("error = %v, want ErrUnreadable", err)
	}
}

// fields is the part of a task the cases about fields state.
type fields struct {
	Text       string
	Priority   string
	Due        string
	DoneOn     string
	Recurrence string
	Projects   []string
	Contexts   []string
	Issues     []Issue
}

func readFields(t *testing.T, line string) fields {
	t.Helper()
	tasks := parse(t, []byte(line))
	if len(tasks) != 1 {
		t.Fatalf("%q: got %d tasks, want 1", line, len(tasks))
	}
	task := tasks[0]
	return fields{
		Text: task.Text, Priority: task.Priority, Due: task.Due, DoneOn: task.DoneOn, Recurrence: task.Recurrence,
		Projects: task.Projects, Contexts: task.Contexts, Issues: task.Issues,
	}
}

func TestParseNoteReadsFieldsInAnyOrderAndInTheMiddleOfTheText(t *testing.T) {
	cases := []struct {
		line string
		want fields
	}{
		{
			"- [ ] write report due:2026-11-20 +work @desk pri:B",
			fields{Text: "write report", Priority: "B", Due: "2026-11-20", Projects: []string{"work"}, Contexts: []string{"desk"}},
		},
		{
			"- [ ] @desk pri:B +work due:2026-11-20 write report",
			fields{Text: "write report", Priority: "B", Due: "2026-11-20", Projects: []string{"work"}, Contexts: []string{"desk"}},
		},
		{"- [ ] call due:2026-11-20 mom", fields{Text: "call mom", Due: "2026-11-20"}},
		{"- [x] file taxes done:2026-11-14", fields{Text: "file taxes", DoneOn: "2026-11-14"}},
		{"- [ ] water plants rec:1w", fields{Text: "water plants", Recurrence: "rec:1w"}},
		{"- [ ] pay rent rec:!2m due:2026-12-01", fields{Text: "pay rent", Recurrence: "rec:!2m", Due: "2026-12-01"}},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			if got := readFields(t, tc.line); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("fields = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestParseNoteReadsThePriorityOfALeadingLetterAndOfAField(t *testing.T) {
	cases := []struct {
		line     string
		priority string
		text     string
	}{
		{"- [ ] (A) first", "A", "first"},
		{"- [ ] second pri:C", "C", "second"},
		{"- [ ] (A) both pri:B", "A", "both"},
		{"- [ ] later (B) in the text", "", "later (B) in the text"},
		{"- [ ] (a) lower case", "", "(a) lower case"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			got := readFields(t, tc.line)
			if got.Priority != tc.priority || got.Text != tc.text {
				t.Errorf("priority = %q, text = %q, want %q and %q", got.Priority, got.Text, tc.priority, tc.text)
			}
		})
	}
}

func TestParseNoteReadsProjectsAndContextsOnceAndNotAnEmailAddress(t *testing.T) {
	got := readFields(t, "- [ ] mail bob@host.example +home +errands +home @phone @phone")

	if !reflect.DeepEqual(got.Projects, []string{"home", "errands"}) || !reflect.DeepEqual(got.Contexts, []string{"phone"}) {
		t.Errorf("projects = %v, contexts = %v", got.Projects, got.Contexts)
	}
	if got.Text != "mail bob@host.example" {
		t.Errorf("text = %q", got.Text)
	}
}

func TestParseNoteReadsNoFieldInsideCodeALinkOrAWikilink(t *testing.T) {
	cases := []string{
		"- [ ] see `due:2026-11-20` for the date",
		"- [ ] see [[note due:2026-11-20]] for the date",
		"- [ ] see [the page](https://a.example/p?due:2026-11-20) for the date",
		"- [ ] see [due:2026-11-20](https://a.example/p) for the date",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			got := readFields(t, line)
			if got.Due != "" || len(got.Issues) != 0 {
				t.Errorf("due = %q, issues = %v, want none", got.Due, got.Issues)
			}
			if want := strings.TrimPrefix(line, "- [ ] "); got.Text != want {
				t.Errorf("text = %q, want %q", got.Text, want)
			}
		})
	}
}

func TestParseNoteKeepsAnImpossibleOrMalformedFieldAsTextWithAnIssue(t *testing.T) {
	cases := []struct {
		line  string
		issue Issue
		text  string
	}{
		{"- [ ] pay due:2026-02-30", IssueInvalidDue, "pay due:2026-02-30"},
		{"- [ ] pay due:2027-02-29", IssueInvalidDue, "pay due:2027-02-29"},
		{"- [ ] pay due:2026-13-01", IssueInvalidDue, "pay due:2026-13-01"},
		{"- [x] pay done:2026-04-31", IssueInvalidDone, "pay done:2026-04-31"},
		{"- [ ] pay rec:2x", IssueInvalidRecurrence, "pay rec:2x"},
		{"- [ ] pay rec:0d", IssueInvalidRecurrence, "pay rec:0d"},
		{"- [ ] pay rec:1000d", IssueInvalidRecurrence, "pay rec:1000d"},
		{"- [ ] pay rec:!w", IssueInvalidRecurrence, "pay rec:!w"},
	}
	for _, tc := range cases {
		t.Run(tc.line, func(t *testing.T) {
			got := readFields(t, tc.line)
			if !reflect.DeepEqual(got.Issues, []Issue{tc.issue}) {
				t.Errorf("issues = %v, want %v", got.Issues, tc.issue)
			}
			if got.Due != "" || got.DoneOn != "" || got.Recurrence != "" {
				t.Errorf("due = %q, doneOn = %q, recurrence = %q, want none", got.Due, got.DoneOn, got.Recurrence)
			}
			if got.Text != tc.text {
				t.Errorf("text = %q, want %q", got.Text, tc.text)
			}
		})
	}
}

func TestParseNoteReadsALeapDay(t *testing.T) {
	got := readFields(t, "- [ ] leap due:2028-02-29")

	if got.Due != "2028-02-29" || len(got.Issues) != 0 {
		t.Errorf("due = %q, issues = %v", got.Due, got.Issues)
	}
}

func TestParseNoteCountsOnlyTheFirstOccurrenceOfAField(t *testing.T) {
	got := readFields(t, "- [ ] x due:2026-11-20 due:2026-11-25")

	if got.Due != "2026-11-20" || got.Text != "x due:2026-11-25" {
		t.Errorf("due = %q, text = %q", got.Due, got.Text)
	}
}

func TestParseNoteLeavesTheRawLineAsItIs(t *testing.T) {
	line := "  >  - [ ]   (A)  keep   due:2026-11-20  spacing ^id-1"
	tasks := parse(t, []byte("> intro\n"+line+"\r\n"))

	if len(tasks) != 1 || tasks[0].Raw != line {
		t.Errorf("tasks = %+v, want one with raw %q", tasks, line)
	}
}

func TestParseNoteReadsTheSameTasksTwice(t *testing.T) {
	note := lines("---", "k: v", "---", "- [ ] a due:2026-11-20 +p @c", "- [x] b done:2026-11-14")

	if first, second := parse(t, note), parse(t, note); !reflect.DeepEqual(first, second) {
		t.Errorf("first = %+v, second = %+v", first, second)
	}
}

// A body made of plain task lines gives one task for each line, whatever the words are.
func FuzzParseNoteFindsEveryPlainTaskLine(f *testing.F) {
	f.Add(uint8(3), "milk eggs")
	f.Add(uint8(1), "a")
	f.Add(uint8(40), "x y z")
	f.Fuzz(func(t *testing.T, count uint8, words string) {
		word := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' {
				return r
			}
			return -1
		}, words)
		if word == "" {
			return
		}
		var body []string
		for n := range int(count) {
			body = append(body, "- [ ] "+word+strings.Repeat("x", n%3))
		}
		tasks, err := ParseNote([]byte(strings.Join(body, "\n")))
		if err != nil {
			t.Fatal(err)
		}
		if len(tasks) != len(body) {
			t.Fatalf("got %d tasks, want %d", len(tasks), len(body))
		}
		for n, task := range tasks {
			if task.Line != n+1 || task.Raw != body[n] {
				t.Fatalf("task %d = line %d raw %q, want line %d raw %q", n, task.Line, task.Raw, n+1, body[n])
			}
		}
	})
}

// Any bytes give tasks whose line and raw text agree with the note, in the order of the lines.
func FuzzParseNoteGivesTasksThatMatchTheirLines(f *testing.F) {
	f.Add([]byte("- [ ] a due:2026-02-30 rec:!1w +p @c (A)\n```\n- [x] b\n```\n> - [-] c\r\n"))
	f.Add([]byte("---\nk: v\n---\n<!-- - [ ] a -->\n- [ ] b"))
	f.Add([]byte("- [ ] a\n"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		tasks, err := ParseNote(raw)
		if err != nil {
			return
		}
		fileLines := strings.Split(string(raw), "\n")
		previous := 0
		for _, task := range tasks {
			if task.Line <= previous || task.Line > len(fileLines) {
				t.Fatalf("line %d out of order or out of range (%d lines)", task.Line, len(fileLines))
			}
			previous = task.Line
			if want := strings.TrimSuffix(fileLines[task.Line-1], "\r"); task.Raw != want {
				t.Fatalf("raw = %q, want the line %q", task.Raw, want)
			}
			if task.Text != strings.TrimSpace(task.Text) || task.Text == "" && task.Raw == "" {
				t.Fatalf("text = %q", task.Text)
			}
		}
	})
}
