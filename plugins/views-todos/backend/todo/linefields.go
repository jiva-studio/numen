package todo

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxRecurrenceCount = 999
	priorityTokenLen   = len("(A)")
)

// lineFields collects what the tokens of a task line say and the tokens that stay in its text.
type lineFields struct {
	priority   string
	due        string
	doneOn     string
	recurrence string
	projects   []string
	contexts   []string
	issues     []Issue
	kept       []string

	seen map[string]bool
}

// readLineFields reads the text after a checkbox. A token that starts inside inline code, a wikilink or
// a Markdown link is text.
func readLineFields(text string) lineFields {
	f := lineFields{seen: map[string]bool{}}
	masked := maskSpans(text)
	for n, token := range splitTokens(text) {
		if masked[token.start] || !f.read(token.text, n == 0) {
			f.kept = append(f.kept, token.text)
		}
	}
	return f
}

type token struct {
	text  string
	start int
}

// splitTokens splits at runs of spaces and tabs.
func splitTokens(text string) []token {
	var tokens []token
	start := -1
	for i := range len(text) {
		isBlank := text[i] == ' ' || text[i] == '\t'
		switch {
		case isBlank && start >= 0:
			tokens = append(tokens, token{text: text[start:i], start: start})
			start = -1
		case !isBlank && start < 0:
			start = i
		}
	}
	if start >= 0 {
		tokens = append(tokens, token{text: text[start:], start: start})
	}
	return tokens
}

// read takes a token as a field and reports whether it was one. An unreadable value is text, with
// an issue.
func (f *lineFields) read(word string, isFirst bool) bool {
	switch {
	case isFirst && isLeadingPriority(word):
		f.priority = word[1:2]
		return true
	case strings.HasPrefix(word, "pri:"):
		return f.readPriority(word)
	case strings.HasPrefix(word, "due:"):
		return f.readDate("due:", word, &f.due, IssueInvalidDue)
	case strings.HasPrefix(word, "done:"):
		return f.readDate("done:", word, &f.doneOn, IssueInvalidDone)
	case strings.HasPrefix(word, "rec:"):
		return f.readRecurrence(word)
	case len(word) > 1 && word[0] == '+':
		f.projects = appendOnce(f.projects, word[1:])
		return true
	case len(word) > 1 && word[0] == '@':
		f.contexts = appendOnce(f.contexts, word[1:])
		return true
	}
	return false
}

func isLeadingPriority(word string) bool {
	return len(word) == priorityTokenLen && word[0] == '(' && word[2] == ')' && isLetter(word[1])
}

func isLetter(c byte) bool { return c >= 'A' && c <= 'Z' }

func appendOnce(list []string, name string) []string {
	for _, one := range list {
		if one == name {
			return list
		}
	}
	return append(list, name)
}

// isFirstOf reports whether the key was not seen before, and marks it as seen.
func (f *lineFields) isFirstOf(key string) bool {
	if f.seen[key] {
		return false
	}
	f.seen[key] = true
	return true
}

func (f *lineFields) readPriority(word string) bool {
	if !f.isFirstOf("pri:") {
		return false
	}
	value := strings.TrimPrefix(word, "pri:")
	if len(value) != 1 || !isLetter(value[0]) {
		f.issues = append(f.issues, IssueInvalidPriority)
		return false
	}
	if f.priority == "" {
		f.priority = value
	}
	return true
}

func (f *lineFields) readDate(key, word string, target *string, issue Issue) bool {
	if !f.isFirstOf(key) {
		return false
	}
	value := strings.TrimPrefix(word, key)
	if !IsDate(value) {
		f.issues = append(f.issues, issue)
		return false
	}
	*target = value
	return true
}

func (f *lineFields) readRecurrence(word string) bool {
	if !f.isFirstOf("rec:") {
		return false
	}
	if !isRecurrence(strings.TrimPrefix(word, "rec:")) {
		f.issues = append(f.issues, IssueInvalidRecurrence)
		return false
	}
	f.recurrence = word
	return true
}

// isRecurrence reports whether the value is an optional !, a count from 1 to 999 and one of d, w, m, y.
func isRecurrence(value string) bool {
	value = strings.TrimPrefix(value, "!")
	if len(value) < 2 || !strings.ContainsRune("dwmy", rune(value[len(value)-1])) {
		return false
	}
	digits := value[:len(value)-1]
	if strings.Trim(digits, "0123456789") != "" {
		return false
	}
	count, err := strconv.Atoi(digits)
	return err == nil && count >= 1 && count <= maxRecurrenceCount
}

// maskSpans marks the bytes of inline code, wikilinks and Markdown links.
func maskSpans(text string) []bool {
	masked := make([]bool, len(text)+1)
	for i := 0; i < len(text); {
		end := spanEnd(text, i)
		if end <= i {
			_, size := utf8.DecodeRuneInString(text[i:])
			i += size
			continue
		}
		for j := i; j < end; j++ {
			masked[j] = true
		}
		i = end
	}
	return masked
}

// spanEnd returns the end of the span that starts at i, or i when none starts there.
func spanEnd(text string, i int) int {
	switch {
	case text[i] == '`':
		return codeEnd(text, i)
	case strings.HasPrefix(text[i:], "[["):
		if end := strings.Index(text[i+2:], "]]"); end >= 0 {
			return i + 2 + end + 2
		}
	case text[i] == '[':
		return linkEnd(text, i)
	}
	return i
}

// codeEnd returns the end of the code span that opens with a run of backticks at i.
func codeEnd(text string, i int) int {
	run := 0
	for i+run < len(text) && text[i+run] == '`' {
		run++
	}
	fence := strings.Repeat("`", run)
	for j := i + run; j < len(text); j++ {
		if !strings.HasPrefix(text[j:], fence) {
			continue
		}
		length := 0
		for j+length < len(text) && text[j+length] == '`' {
			length++
		}
		if length == run {
			return j + run
		}
		j += length - 1
	}
	return i
}

// linkEnd returns the end of the Markdown link [text](destination) that starts at i.
func linkEnd(text string, i int) int {
	closing := strings.IndexByte(text[i:], ']')
	if closing < 0 || i+closing+1 >= len(text) || text[i+closing+1] != '(' {
		return i
	}
	paren := strings.IndexByte(text[i+closing:], ')')
	if paren < 0 {
		return i
	}
	return i + closing + paren + 1
}
