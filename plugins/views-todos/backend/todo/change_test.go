package todo

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const tickDay = "2026-11-14"

func TestChangeLineTicksAnOpenTask(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "the box gets an x and done: today is added at the end",
			line: "- [ ] buy milk",
			want: "- [x] buy milk done:2026-11-14",
		},
		{
			name: "done: is added before a block id at the end, so the anchor stays last",
			line: "- [ ] buy milk ^milk-1",
			want: "- [x] buy milk done:2026-11-14 ^milk-1",
		},
		{
			name: "an existing done: gets today in place",
			line: "- [ ] water plants done:2026-01-01 +home",
			want: "- [x] water plants done:2026-11-14 +home",
		},
		{
			name: "only the first done: is changed and a second one is text",
			line: "- [ ] a done:2026-01-01 done:2026-02-02",
			want: "- [x] a done:2026-11-14 done:2026-02-02",
		},
		{
			name: "an invalid done: is text and today is added at the end",
			line: "- [ ] a done:2026-02-30",
			want: "- [x] a done:2026-02-30 done:2026-11-14",
		},
		{
			name: "a done: inside inline code or a wikilink is text",
			line: "- [ ] see `done:2026-01-01` and [[done:2026-01-02]]",
			want: "- [x] see `done:2026-01-01` and [[done:2026-01-02]] done:2026-11-14",
		},
		{
			name: "the quote prefix, the indentation, the number marker and the fields are kept",
			line: "> \t  12) [ ] (A) call @phone due:2026-11-20 pri:B owner:me",
			want: "> \t  12) [x] (A) call @phone due:2026-11-20 pri:B owner:me done:2026-11-14",
		},
		{
			name: "the spacing of the line is kept",
			line: "*   [ ]   a    b\tc",
			want: "*   [x]   a    b\tc done:2026-11-14",
		},
		{
			name: "a callout task is ticked inside its quote",
			line: ">> - [ ] in a callout",
			want: ">> - [x] in a callout done:2026-11-14",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, change, err := ChangeLine(tc.line, true, tickDay)
			if err != nil {
				t.Fatalf("ChangeLine(%q) error = %v", tc.line, err)
			}
			if got != tc.want || change != ChangeCompleted {
				t.Errorf("ChangeLine(%q) = %q %q, want %q %q", tc.line, got, change, tc.want, ChangeCompleted)
			}
		})
	}
}

func TestChangeLineUnticksATickedTask(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "the box gets a space and done: with the space before it is removed",
			line: "- [x] buy milk done:2026-11-01 +home",
			want: "- [ ] buy milk +home",
		},
		{
			name: "an upper case X is unticked",
			line: "- [X] buy milk",
			want: "- [ ] buy milk",
		},
		{
			name: "a task of another status is unticked",
			line: "- [-] cancelled one",
			want: "- [ ] cancelled one",
		},
		{
			name: "done: before a block id is removed and the anchor stays",
			line: "- [x] buy milk done:2026-11-14 ^milk-1",
			want: "- [ ] buy milk ^milk-1",
		},
		{
			name: "done: as the first token is removed with the space before it",
			line: "- [x] done:2026-11-01 a",
			want: "- [ ] a",
		},
		{
			name: "an invalid done: is text and stays",
			line: "- [x] a done:2026-02-30",
			want: "- [ ] a done:2026-02-30",
		},
		{
			name: "a recurring task keeps its due: and rec:",
			line: "- [x] review due:2026-11-01 rec:1w",
			want: "- [ ] review due:2026-11-01 rec:1w",
		},
		{
			name: "the quote prefix and the indentation are kept",
			line: ">   - [x] quoted done:2026-11-01",
			want: ">   - [ ] quoted",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, change, err := ChangeLine(tc.line, false, tickDay)
			if err != nil {
				t.Fatalf("ChangeLine(%q) error = %v", tc.line, err)
			}
			if got != tc.want || change != ChangeReopened {
				t.Errorf("ChangeLine(%q) = %q %q, want %q %q", tc.line, got, change, tc.want, ChangeReopened)
			}
		})
	}
}

func TestChangeLineRefusesAChangeThatDoesNotFitTheTask(t *testing.T) {
	cases := []struct {
		name   string
		line   string
		isDone bool
		want   error
	}{
		{"a tick of a done task", "- [x] a", true, ErrNotOpen},
		{"a tick of a task done with an upper case X", "- [X] a", true, ErrNotOpen},
		{"a tick of a cancelled task", "- [-] a", true, ErrNotOpen},
		{"a tick of a task with a character that is not ASCII", "- [→] a", true, ErrNotOpen},
		{"a tick of a task with an invalid recurrence", "- [ ] a rec:0d", true, ErrInvalidRecurrence},
		{"an untick of an open task", "- [ ] a", false, ErrNotDone},
		{"a line with no checkbox", "- a", true, ErrNotTask},
		{"a checkbox with no text", "- [ ]", true, ErrNotTask},
		{"a checkbox with no list marker", "[ ] a", true, ErrNotTask},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ChangeLine(tc.line, tc.isDone, tickDay)
			if !errors.Is(err, tc.want) {
				t.Errorf("ChangeLine(%q, %v) error = %v, want %v", tc.line, tc.isDone, err, tc.want)
			}
		})
	}
}

// insertsDone reports whether ticked is before with " done:<today>" put in at one place.
func insertsDone(before, ticked string) bool {
	field := " done:" + tickDay
	for at := strings.Index(ticked, field); at >= 0; {
		if ticked[:at]+ticked[at+len(field):] == before {
			return true
		}
		next := strings.Index(ticked[at+1:], field)
		if next < 0 {
			break
		}
		at += 1 + next
	}
	return false
}

// replacesDone reports whether ticked is before with the value of one done: token set to today.
func replacesDone(before, ticked string) bool {
	if len(before) != len(ticked) {
		return false
	}
	for at := strings.Index(before, "done:"); at >= 0; {
		end := at + len("done:") + len(tickDay)
		if end <= len(before) && IsDate(before[at+len("done:"):end]) &&
			before[:at] == ticked[:at] && ticked[at:end] == "done:"+tickDay && before[end:] == ticked[end:] {
			return true
		}
		next := strings.Index(before[at+1:], "done:")
		if next < 0 {
			break
		}
		at += 1 + next
	}
	return false
}

// For any line, a tick changes the checkbox space to x and either adds done:today or sets the
// value of one done: token; no other byte changes, and the parsed task keeps its text and fields.
// An untick of a line with no done: before the tick gives the line back.
func FuzzChangeLineChangesOnlyTheCheckboxAndTheCompletionDate(f *testing.F) {
	f.Add("- [ ] buy milk")
	f.Add("- [ ] buy milk ^milk-1")
	f.Add("> \t  12) [ ] (A) call @phone due:2026-11-20 pri:B")
	f.Add("- [ ] a done:2026-01-01 +home")
	f.Add("* [ ] see `done:2026-01-01` [[x]] [l](done:2026-01-01)")
	f.Add("- [x] already")
	f.Add("\ufeff- [ ] first line")
	f.Fuzz(func(t *testing.T, line string) {
		if strings.ContainsAny(line, "\r\n") {
			return
		}
		before, err := ParseNote([]byte(line))
		if err != nil || len(before) != 1 || before[0].Status != StatusOpen || strings.Contains(line, "rec:") {
			return
		}
		ticked, change, err := ChangeLine(line, true, tickDay)
		if err != nil {
			t.Fatalf("ChangeLine(%q) error = %v", line, err)
		}
		if change != ChangeCompleted {
			t.Fatalf("ChangeLine(%q) change = %q", line, change)
		}
		box := strings.Index(line, "[ ]")
		if box < 0 || len(ticked) <= box+1 || ticked[box+1] != 'x' {
			t.Fatalf("ChangeLine(%q) = %q: the box does not hold x", line, ticked)
		}
		withX := line[:box+1] + "x" + line[box+2:]
		if !insertsDone(withX, ticked) && !replacesDone(withX, ticked) {
			t.Fatalf("ChangeLine(%q) = %q: a byte outside the box and done: changed", line, ticked)
		}
		after, err := ParseNote([]byte(ticked))
		if err != nil || len(after) != 1 {
			t.Fatalf("the ticked line %q is not one task: %v", ticked, err)
		}
		got, want := after[0], before[0]
		if got.Status != StatusDone {
			t.Fatalf("ticked %q parses with status %q", ticked, got.Status)
		}
		if slices.Contains(want.Issues, IssueInvalidDone) {
			// The added done: follows an unreadable one, so it is text.
			return
		}
		if got.Text != want.Text || got.Due != want.Due ||
			got.Priority != want.Priority || !slices.Equal(got.Projects, want.Projects) || !slices.Equal(got.Contexts, want.Contexts) {
			t.Fatalf("ticked %q parses as %+v, was %+v", ticked, got, want)
		}
		if strings.Contains(line, "done:") || strings.TrimRight(line, " \t") != line {
			return
		}
		if got.DoneOn != tickDay {
			t.Fatalf("ticked %q has done %q, want %q", ticked, got.DoneOn, tickDay)
		}
		unticked, _, err := ChangeLine(ticked, false, tickDay)
		if err != nil || unticked != line {
			t.Fatalf("untick of %q = %q %v, want the line %q", ticked, unticked, err, line)
		}
	})
}
