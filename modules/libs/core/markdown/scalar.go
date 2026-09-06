// A scalar of the frontmatter: the bytes its token occupies, and the spelling
// YAML has to give a value written in its place.

package markdown

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// scalarSpan is the bytes one scalar occupies, so that changing a value leaves
// everything else on its line — a trailing comment, the spacing, the other keys
// of the entry — exactly where it was.
//
// The span covers the whole token, quotes and all.
//
// A scalar written over lines of its own — with `|`, with `>`, or a quoted one
// carried across a line break — is not one token on one line. Its link stays as
// it was written and shows as a problem.
func (d *Document) scalarSpan(node *yaml.Node) (start, end int, ok bool) {
	if node.Kind != yaml.ScalarNode {
		return 0, 0, false
	}
	lines := lineOffsets(d.front)
	if node.Line < 1 || node.Line >= len(lines) || node.Column < 1 {
		return 0, 0, false
	}
	start = columnOffset(d.front, lines, node.Line, node.Column)
	if start >= len(d.front) {
		return 0, 0, false
	}

	switch {
	case node.Style == 0:
		end = start + len(node.Value)
		if end > len(d.front) || string(d.front[start:end]) != node.Value {
			return 0, 0, false
		}
	case node.Style&yaml.SingleQuotedStyle != 0:
		if end, ok = quotedEnd(d.front, start, '\''); !ok {
			return 0, 0, false
		}
	case node.Style&yaml.DoubleQuotedStyle != 0:
		if end, ok = quotedEnd(d.front, start, '"'); !ok {
			return 0, 0, false
		}
	default:
		return 0, 0, false
	}

	// What the token says, read back. A span that does not say what the node
	// said is the wrong span, and nothing is written over.
	var said string
	if err := yaml.Unmarshal(d.front[start:end], &said); err != nil || said != node.Value {
		return 0, 0, false
	}
	return start, end, true
}

// quotedEnd is where a quoted scalar ends, counting from the quote it opens
// with. Inside a single-quoted one a doubled quote is a quote; inside a
// double-quoted one a backslash escapes what follows.
func quotedEnd(front []byte, start int, quote byte) (int, bool) {
	if front[start] != quote {
		return 0, false
	}
	for at := start + 1; at < len(front); at++ {
		if breakWidth(front, at) > 0 {
			return 0, false
		}
		switch front[at] {
		case '\\':
			if quote == '"' && at+1 < len(front) && breakWidth(front, at+1) == 0 {
				at++
			}
		case quote:
			if quote == '\'' && at+1 < len(front) && front[at+1] == '\'' {
				at++
				continue
			}
			return at + 1, true
		}
	}
	return 0, false
}

// scalar is a value as YAML has to spell it, so that a name carrying `[`, `#`,
// `&` or a word YAML reads as a number goes into a file as the text it is.
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

// scalarLike is a value spelled the way the value it replaces was spelled. How
// somebody quotes their own frontmatter is theirs. A style the value cannot be
// written in is written as YAML has to spell it.
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
	written := strings.TrimRight(out.String(), "\n")

	var said string
	if err := yaml.Unmarshal([]byte(written), &said); err != nil || said != value {
		return scalar(value)
	}
	return written, nil
}
