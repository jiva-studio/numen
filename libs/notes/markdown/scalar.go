// A scalar of the frontmatter: the bytes its token occupies, and the spelling
// YAML has to give a value written in its place.

package markdown

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// scalarSpan returns the byte range one scalar occupies. Changing a value in that
// range leaves the rest of its line unchanged: a trailing comment, the spacing, and the
// other keys of the entry.
//
// The span covers the whole token, including quotes.
//
// A scalar spanning several lines, with `|`, with `>`, or a quoted scalar
// continued across a line break, is not one token on one line. Its link is left
// unchanged and reported as a problem.
func (d *Document) scalarSpan(node *yaml.Node) (start, end int, ok bool) {
	start, ok = d.readScalarStart(node)
	if !ok {
		return 0, 0, false
	}
	end, ok = d.findScalarEnd(node, start)
	if !ok {
		return 0, 0, false
	}
	if !isValidScalarSpan(d.front, start, end, node.Value) {
		return 0, 0, false
	}
	return start, end, true
}

func (d *Document) readScalarStart(node *yaml.Node) (int, bool) {
	if node.Kind != yaml.ScalarNode {
		return 0, false
	}
	lines := lineOffsets(d.front)
	if node.Line < 1 || node.Line >= len(lines) || node.Column < 1 {
		return 0, false
	}
	start := columnOffset(d.front, lines, node.Line, node.Column)
	if start >= len(d.front) {
		return 0, false
	}
	return start, true
}

func (d *Document) findScalarEnd(node *yaml.Node, start int) (int, bool) {
	switch {
	case node.Style == 0:
		end := start + len(node.Value)
		if end > len(d.front) || string(d.front[start:end]) != node.Value {
			return 0, false
		}
		return end, true
	case node.Style&yaml.SingleQuotedStyle != 0:
		return findQuotedEnd(d.front, start, '\'')
	case node.Style&yaml.DoubleQuotedStyle != 0:
		return findQuotedEnd(d.front, start, '"')
	default:
		return 0, false
	}
}

func isValidScalarSpan(front []byte, start, end int, want string) bool {
	var parsed string
	if err := yaml.Unmarshal(front[start:end], &parsed); err != nil || parsed != want {
		return false
	}
	return true
}

func canEscapeBackslash(front []byte, offset int, quote byte) bool {
	return quote == '"' && offset+1 < len(front) && breakWidth(front, offset+1) == 0
}

func isEscapedSingleQuote(front []byte, offset int, quote byte) bool {
	return quote == '\'' && offset+1 < len(front) && front[offset+1] == '\''
}

// findQuotedEnd is where a quoted scalar ends, counting from the quote it opens
// with. Inside a single-quoted one a doubled quote is a quote; inside a
// double-quoted one a backslash escapes what follows.
func findQuotedEnd(front []byte, start int, quote byte) (int, bool) {
	if front[start] != quote {
		return 0, false
	}
	for pos := start + 1; pos < len(front); pos++ {
		if breakWidth(front, pos) > 0 {
			return 0, false
		}
		switch front[pos] {
		case '\\':
			if canEscapeBackslash(front, pos, quote) {
				pos++
			}
		case quote:
			if isEscapedSingleQuote(front, pos, quote) {
				pos++
				continue
			}
			return pos + 1, true
		}
	}
	return 0, false
}

// scalar returns a value spelled as YAML requires. A name containing `[`, `#`,
// `&` or a word YAML parses as a number is written to the file as text.
func scalar(value string) (string, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(value); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

// scalarLike returns a value spelled in the style of the value it replaces. The quoting
// style of the user's frontmatter is preserved. A style the value cannot be
// written in falls back to the YAML spelling.
func scalarLike(value string, style yaml.Style) (string, error) {
	if style == 0 {
		return scalar(value)
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&yaml.Node{Kind: yaml.ScalarNode, Style: style, Value: value}); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	content := strings.TrimRight(out.String(), "\n")

	var parsed string
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil || parsed != value {
		return scalar(value)
	}
	return content, nil
}
