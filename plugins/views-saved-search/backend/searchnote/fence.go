package searchnote

import "strings"

// Language is the info string of the fenced block that holds the query.
const Language = "numen-query"

const (
	minFenceLength = 3
	maxFenceIndent = 3
)

// line is one line of a body: its text without the line ending, the offset of its first byte and the
// offset after its line ending.
type line struct {
	text       string
	start, end int
}

// block is a fenced code block of a body: the lines of its fences, its info string, and whether a fence
// closes it.
type block struct {
	opening  line
	closing  line
	info     string
	mark     byte
	length   int
	isClosed bool
	content  []line
}

// fence is the opening or closing line of a fenced code block.
type fence struct {
	mark   byte
	length int
	info   string
}

func splitLines(body string) []line {
	var lines []line
	for start := 0; start < len(body); {
		end := strings.IndexByte(body[start:], '\n')
		next := len(body)
		if end >= 0 {
			next = start + end + 1
		}
		text := strings.TrimSuffix(strings.TrimSuffix(body[start:next], "\n"), "\r")
		lines = append(lines, line{text: text, start: start, end: next})
		start = next
	}
	return lines
}

func readFence(text string) (fence, bool) {
	trimmed := strings.TrimLeft(text, " ")
	if len(text)-len(trimmed) > maxFenceIndent || trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return fence{mark: 0, length: 0, info: ""}, false
	}
	mark := trimmed[0]
	length := len(trimmed) - len(strings.TrimLeft(trimmed, string(mark)))
	info := strings.TrimSpace(trimmed[length:])
	if length < minFenceLength || (mark == '`' && strings.Contains(info, "`")) {
		return fence{mark: 0, length: 0, info: ""}, false
	}
	return fence{mark: mark, length: length, info: info}, true
}

func isClosing(text string, open fence) bool {
	closing, isFence := readFence(text)
	return isFence && closing.mark == open.mark && closing.length >= open.length && closing.info == ""
}

func findBlocks(lines []line) []block {
	var blocks []block
	for i := 0; i < len(lines); i++ {
		open, isFence := readFence(lines[i].text)
		if !isFence {
			continue
		}
		found := block{
			opening: lines[i], closing: line{text: "", start: 0, end: 0}, info: open.info, mark: open.mark,
			length: open.length, isClosed: false, content: nil,
		}
		for i++; i < len(lines); i++ {
			if isClosing(lines[i].text, open) {
				found.closing, found.isClosed = lines[i], true
				break
			}
			found.content = append(found.content, lines[i])
		}
		blocks = append(blocks, found)
	}
	return blocks
}

func findQueryBlocks(body string) []block {
	var found []block
	for _, one := range findBlocks(splitLines(body)) {
		if one.info == Language {
			found = append(found, one)
		}
	}
	return found
}

func (b block) getText() string {
	texts := make([]string, len(b.content))
	for i, one := range b.content {
		texts[i] = one.text
	}
	return strings.Join(texts, "\n")
}

func getFenceLength(query string, mark byte) int {
	length := minFenceLength
	for one := range strings.SplitSeq(query, "\n") {
		trimmed := strings.TrimLeft(one, " ")
		run := len(trimmed) - len(strings.TrimLeft(trimmed, string(mark)))
		length = max(length, run+1)
	}
	return length
}

func formatBlock(query string) string {
	fence := strings.Repeat("`", getFenceLength(query, '`'))
	return fence + Language + "\n" + query + "\n" + fence + "\n"
}
