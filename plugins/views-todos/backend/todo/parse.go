package todo

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

const (
	byteOrderMark   = "\ufeff"
	codeIndent      = 4
	tabWidth        = 4
	depthWidth      = 2
	maxMarkerDigits = 9
)

// ParseNote returns the tasks of a note in the order of its lines. A note whose frontmatter cannot
// be read or is never closed has no tasks and gives ErrUnreadable.
func ParseNote(raw []byte) ([]Task, error) {
	doc, err := markdown.Open(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	body := doc.Body()
	start := len(raw) - len(body)
	bomLength := 0
	if strings.HasPrefix(string(raw), byteOrderMark) {
		bomLength = len(byteOrderMark)
	}
	if start == bomLength {
		// No frontmatter: the first line keeps its byte order mark in the raw line.
		start = 0
		if markdown.IsFrontmatterStart(string(raw)) {
			return nil, ErrUnreadable
		}
	}
	firstLine := strings.Count(string(raw[:start]), "\n") + 1
	scanner := newScanner()
	var tasks []Task
	for n, line := range strings.Split(string(raw[start:]), "\n") {
		line = strings.TrimSuffix(line, "\r")
		content := line
		if n == 0 && start == 0 {
			content = strings.TrimPrefix(line, byteOrderMark)
		}
		if found, ok := scanner.scan(content, firstLine+n, line); ok {
			tasks = append(tasks, found)
		}
	}
	return tasks, nil
}

// scanner follows the lines of a body: the code fence, the HTML comment and the open list.
type scanner struct {
	fence     markdown.Fence
	isComment bool
	isList    bool
}

func newScanner() *scanner { return &scanner{} }

// scan reads one line. Raw is the line as the file has it; content is the text to read.
func (s *scanner) scan(content string, number int, raw string) (Task, bool) {
	if s.skips(content) {
		return Task{}, false
	}
	rest := stripQuote(content)
	indent, rest := splitIndent(rest)
	if strings.TrimSpace(rest) == "" {
		return Task{}, false
	}
	marker := getMarkerLength(rest)
	if marker == 0 {
		if indent == 0 {
			s.isList = false
		}
		return Task{}, false
	}
	if indent >= codeIndent && !s.isList {
		return Task{}, false
	}
	s.isList = true
	return readTask(rest[marker:], raw, number, indent)
}

// skips reports whether the line is inside a code fence or an HTML comment, or opens or closes
// one, so it holds no task.
func (s *scanner) skips(line string) bool {
	if s.isComment {
		if strings.Contains(line, "-->") {
			s.isComment = false
		}
		return true
	}
	if s.fence.IsCrossedBy(line) || s.fence.IsInside() {
		return true
	}
	if opening := strings.TrimSpace(line); strings.HasPrefix(opening, "<!--") {
		s.isComment = !strings.Contains(opening[len("<!--"):], "-->")
		return true
	}
	return false
}

// stripQuote removes the block-quote markers at the start of a line.
func stripQuote(line string) string {
	for {
		trimmed := strings.TrimLeft(line, " \t")
		after, found := strings.CutPrefix(trimmed, ">")
		if !found {
			return line
		}
		line = after
	}
}

// splitIndent returns the width of the spaces and tabs that start the line and the rest.
func splitIndent(line string) (int, string) {
	width := 0
	for i := range len(line) {
		switch line[i] {
		case ' ':
			width++
		case '\t':
			width += tabWidth
		default:
			return width, line[i:]
		}
	}
	return width, ""
}

// getMarkerLength returns the length of a list marker and the spaces after it, or 0 when the line
// does not start with a list item.
func getMarkerLength(rest string) int {
	length := getNumberMarkerLength(rest)
	if rest != "" && strings.IndexByte("-*+", rest[0]) >= 0 {
		length = 1
	}
	if length == 0 {
		return 0
	}
	spaces := len(rest[length:]) - len(strings.TrimLeft(rest[length:], " "))
	if spaces == 0 {
		return 0
	}
	return length + spaces
}

// readTask reads what follows a list marker as a checkbox and a text.
func readTask(afterMarker, raw string, number, indent int) (Task, bool) {
	box, ok := strings.CutPrefix(afterMarker, "[")
	if !ok || box == "" {
		return Task{}, false
	}
	mark, size := utf8.DecodeRuneInString(box)
	text, ok := strings.CutPrefix(box[size:], "]")
	if !ok || mark == ']' {
		return Task{}, false
	}
	if text != "" && text[0] != ' ' && text[0] != '\t' {
		return Task{}, false
	}
	text = strings.TrimLeft(text, " \t")
	if strings.TrimSpace(text) == "" {
		return Task{}, false
	}
	found := readLineFields(text)
	return Task{
		Path:       "",
		Line:       number,
		Raw:        raw,
		Text:       strings.TrimSpace(strings.Join(found.kept, " ")),
		Status:     getStatus(mark),
		Priority:   found.priority,
		Due:        found.due,
		DoneOn:     found.doneOn,
		Recurrence: found.recurrence,
		Projects:   found.projects,
		Contexts:   found.contexts,
		Issues:     found.issues,
		Depth:      indent / depthWidth,
	}, true
}

func getStatus(mark rune) Status {
	switch mark {
	case ' ':
		return StatusOpen
	case 'x', 'X':
		return StatusDone
	}
	return StatusOther
}

// getNumberMarkerLength returns the length of a number of 1 to 9 digits and its dot or parenthesis,
// or 0 when the line starts with none.
func getNumberMarkerLength(rest string) int {
	digits := 0
	for digits < len(rest) && rest[digits] >= '0' && rest[digits] <= '9' {
		digits++
	}
	if digits == 0 || digits > maxMarkerDigits || digits >= len(rest) {
		return 0
	}
	if rest[digits] != '.' && rest[digits] != ')' {
		return 0
	}
	return digits + 1
}
