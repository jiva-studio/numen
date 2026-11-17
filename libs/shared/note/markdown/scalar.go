// A scalar of the frontmatter: the bytes its token occupies, and the spelling
// YAML has to give a value written in its place.

package markdown

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

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
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="breaking from the switch at the end of the loop body advances to the next iteration identically to continue"
				continue
			}
			return pos + 1, true
		}
	}
	return 0, false
}

func scalar(value string) (string, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="SetIndent does not affect encoding of a single scalar string"
	enc.SetIndent(2)
	// gomutants:disable-next-line BRANCH_IF reason="encoding a Go string to bytes.Buffer cannot fail"
	if err := enc.Encode(value); err != nil {
		return "", err
	}
	// gomutants:disable-next-line BRANCH_IF reason="closing a yaml.Encoder backed by bytes.Buffer cannot fail"
	if err := enc.Close(); err != nil {
		return "", err
	}
	return strings.TrimRight(out.String(), "\n"), nil
}

func scalarLike(value string, style yaml.Style) (string, error) {
	if style == 0 {
		return scalar(value)
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	// gomutants:disable-next-line STATEMENT_REMOVE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="SetIndent does not affect encoding of a single scalar node"
	enc.SetIndent(2)
	// gomutants:disable-next-line BRANCH_IF reason="encoding a scalar yaml.Node to bytes.Buffer cannot fail"
	if err := enc.Encode(&yaml.Node{Kind: yaml.ScalarNode, Style: style, Value: value}); err != nil {
		return "", err
	}
	// gomutants:disable-next-line BRANCH_IF reason="closing a yaml.Encoder backed by bytes.Buffer cannot fail"
	if err := enc.Close(); err != nil {
		return "", err
	}
	content := strings.TrimRight(out.String(), "\n")

	var parsed string
	// gomutants:disable-next-line INVERT_LOGICAL reason="yaml.Encoder output for a scalar node is always valid YAML so err is always nil"
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil || parsed != value {
		return scalar(value)
	}
	return content, nil
}
