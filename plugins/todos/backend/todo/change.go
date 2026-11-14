package todo

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"
)

// Change is what ChangeLine did to a task line.
type Change string

const (
	// ChangeCompleted is an open task that was ticked: its box holds x and it has done: today.
	ChangeCompleted Change = "completed"
	// ChangeRescheduled is an open recurring task that was ticked: its due: moved and it stays open.
	ChangeRescheduled Change = "rescheduled"
	// ChangeReopened is a task that was unticked: its box holds a space and its done: is gone.
	ChangeReopened Change = "reopened"
)

var (
	// ErrNotTask is a line that is not a task line.
	ErrNotTask = errors.New("the line is not a task")
	// ErrNotOpen is a tick of a task that is done or has another status.
	ErrNotOpen = errors.New("the task is not open")
	// ErrNotDone is an untick of a task that is open.
	ErrNotDone = errors.New("the task is not done")
	// ErrInvalidRecurrence is a tick of a task whose rec: cannot be read. A tick of a task with a
	// valid rec: gives it too, until ticking moves the due date of a recurring task.
	ErrInvalidRecurrence = errors.New("the recurrence of the task cannot be read")
)

const doneKey = "done:"

// taskLine is a task line cut at its checkbox: the byte offset of the checkbox character and its
// size, the offset of the text after the checkbox, and the task the line holds.
type taskLine struct {
	line      string
	mark      int
	markSize  int
	textStart int
	task      Task
}

// ChangeLine returns the line after the task is ticked (isDone) or unticked at today, a
// YYYY-MM-DD date, and the change kind. Only the checkbox character and the done: token change;
// every other byte of the line is kept.
func ChangeLine(line string, isDone bool, today string) (string, Change, error) {
	cut, err := cutTaskLine(line)
	if err != nil {
		return line, "", err
	}
	if isDone {
		return tick(cut, today)
	}
	return untick(cut)
}

func cutTaskLine(line string) (taskLine, error) {
	content := strings.TrimPrefix(line, byteOrderMark)
	rest := stripQuote(content)
	indent, rest := splitIndent(rest)
	marker := getMarkerLength(rest)
	if marker == 0 {
		return taskLine{}, ErrNotTask
	}
	afterMarker := rest[marker:]
	task, ok := readTask(afterMarker, line, 1, indent)
	if !ok {
		return taskLine{}, ErrNotTask
	}
	box := len(line) - len(afterMarker)
	_, size := utf8.DecodeRuneInString(line[box+1:])
	afterBox := line[box+1+size+1:]
	textStart := len(line) - len(strings.TrimLeft(afterBox, " \t"))
	return taskLine{line: line, mark: box + 1, markSize: size, textStart: textStart, task: task}, nil
}

func setMark(line string, c taskLine, mark string) string {
	return line[:c.mark] + mark + line[c.mark+c.markSize:]
}

func tick(c taskLine, today string) (string, Change, error) {
	if c.task.Status != StatusOpen {
		return c.line, "", ErrNotOpen
	}
	if c.task.Recurrence != "" || slices.Contains(c.task.Issues, IssueInvalidRecurrence) {
		return c.line, "", ErrInvalidRecurrence
	}
	line := c.line
	if start, end, found := c.findDoneToken(); found {
		line = line[:start] + doneKey + today + line[end:]
	} else {
		at := c.getAppendOffset()
		line = line[:at] + " " + doneKey + today + line[at:]
	}
	return setMark(line, c, "x"), ChangeCompleted, nil
}

func untick(c taskLine) (string, Change, error) {
	if c.task.Status == StatusOpen {
		return c.line, "", ErrNotDone
	}
	line := c.line
	isRecurring := c.task.Recurrence != "" || slices.Contains(c.task.Issues, IssueInvalidRecurrence)
	if start, end, found := c.findDoneToken(); found && !isRecurring {
		// The blank before the token goes with it; readTask puts one before the first token too.
		line = line[:start-1] + line[end:]
	}
	return setMark(line, c, " "), ChangeReopened, nil
}

func (c taskLine) findDoneToken() (int, int, bool) {
	// The first done: token outside code and links is the one ParseNote reads, and only when its
	// value is a date.
	text := c.line[c.textStart:]
	masked := maskSpans(text)
	for _, one := range splitTokens(text) {
		if masked[one.start] || !strings.HasPrefix(one.text, doneKey) {
			continue
		}
		if !IsDate(strings.TrimPrefix(one.text, doneKey)) {
			return 0, 0, false
		}
		start := c.textStart + one.start
		return start, start + len(one.text), true
	}
	return 0, 0, false
}

func (c taskLine) getAppendOffset() int {
	// An added field goes after the last token, before the trailing blanks and a trailing ^block-id.
	end := len(strings.TrimRight(c.line, " \t"))
	tokens := splitTokens(c.line[c.textStart:end])
	if len(tokens) == 0 {
		return end
	}
	last := tokens[len(tokens)-1]
	if !isBlockID(last.text) {
		return end
	}
	return len(strings.TrimRight(c.line[:c.textStart+last.start], " \t"))
}

func isBlockID(word string) bool {
	id, found := strings.CutPrefix(word, "^")
	if !found || id == "" {
		return false
	}
	return strings.Trim(id, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-") == ""
}
