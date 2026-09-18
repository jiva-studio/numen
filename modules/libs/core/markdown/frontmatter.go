// The frontmatter block, written one key at a time: the lines a key occupies,
// what is put in their place, and the blocks no key of which can be replaced
// alone.

package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// render is one node as YAML, ending with the break every block of it ends
// with.
func render(node *yaml.Node) (string, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// put writes one top-level key and what it holds, keeping the comment written
// on the key's own line. That comment is the person's, on a key the application
// owns as much as on any other.
func (d *Document) put(key string, value *yaml.Node) error {
	node, err := d.getWritableNode()
	if err != nil {
		return err
	}
	name := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	pair(node, key).carry(name, value)
	rendered, err := render(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{name, value}})
	if err != nil {
		return err
	}
	return d.set(key, []byte(strings.ReplaceAll(rendered, "\n", d.eol)))
}

// held is the two nodes one key of a mapping stands as, and is empty where the
// mapping has no such key.
type held struct{ name, value *yaml.Node }

// pair is the two nodes one key of a mapping stands as.
func pair(node *yaml.Node, key string) held {
	if node == nil || node.Kind != yaml.MappingNode {
		return held{}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return held{name: node.Content[i], value: node.Content[i+1]}
		}
	}
	return held{}
}

// valueOf is the node one key of an entry holds, or nil when it has no such key.
func valueOf(item *yaml.Node, key string) *yaml.Node {
	if item.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(item.Content); i += 2 {
		if item.Content[i].Value == key {
			return item.Content[i+1]
		}
	}
	return nil
}

// carry puts the comment written on a key's own line onto the nodes replacing
// it. A mapping or a list carries it on the key and a scalar on the value, so
// both are read off both.
func (h held) carry(name, value *yaml.Node) {
	if h.name == nil {
		return
	}
	name.LineComment = h.name.LineComment
	if value != h.value {
		value.LineComment = h.value.LineComment
	}
}

// set replaces the lines one top-level key occupies, or appends them when the
// key is not there yet. Empty replacement removes the key.
func (d *Document) set(key string, rendered []byte) error {
	node, err := d.getWritableNode()
	if err != nil {
		return err
	}
	rendered = indentLines(rendered, d.indent(node))

	start, end, found := d.span(node, key)
	if !found {
		if len(rendered) == 0 {
			return nil
		}
		front := append([]byte(nil), d.front...)
		if len(front) > 0 && !endsWithBreak(front) {
			front = append(front, d.eol...)
		}
		if err := d.commit(append(front, rendered...)); err != nil {
			return err
		}
		if node == nil && len(d.open) == 0 {
			// A note with no frontmatter grows one.
			d.open = []byte("---" + d.eol)
			d.shut = []byte("---" + d.eol)
		}
		return nil
	}

	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	return d.commit(append(front, d.front[end:]...))
}

// commit puts a frontmatter block in place of the one standing. A block that
// comes out of a splice unreadable is not written, and the note keeps the bytes
// it arrived as.
func (d *Document) commit(front []byte) error {
	was := d.front
	d.front = front
	if _, err := d.readMapping(); err != nil {
		d.front = was
		return err
	}
	return nil
}

// splice puts bytes in place of a range of the frontmatter.
func (d *Document) splice(start, end int, rendered []byte) {
	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	d.front = append(front, d.front[end:]...)
}

// indent is the whitespace the frontmatter's own keys stand at. A block written
// in from the margin is a mapping that ends at the first line written flush,
// and the keys below it become text.
func (d *Document) indent(node *yaml.Node) string {
	if node == nil || len(node.Content) == 0 {
		return ""
	}
	lines := lineOffsets(d.front)
	at := node.Content[0].Line
	if at < 1 || at >= len(lines) {
		return ""
	}
	return getIndent(string(d.front[lines[at-1]:lines[at]]))
}

// indentLines is rendered lines written at the indentation the block's keys
// stand at. A line with nothing on it takes none.
func indentLines(rendered []byte, indent string) []byte {
	if indent == "" || len(rendered) == 0 {
		return rendered
	}
	out := make([]byte, 0, len(rendered)+4*len(indent))
	for at := 0; at < len(rendered); {
		end := len(rendered)
		if next := bytes.IndexByte(rendered[at:], '\n'); next >= 0 {
			end = at + next + 1
		}
		if len(bytes.TrimSpace(rendered[at:end])) > 0 {
			out = append(out, indent...)
		}
		out = append(out, rendered[at:end]...)
		at = end
	}
	return out
}

// span is the byte range one top-level key occupies in the frontmatter,
// including the lines its value continues onto.
//
// It ends at the last line of the key's own value, so a comment, a blank line,
// or bytes standing inside the delimiters under no key at all are left where
// they are.
func (d *Document) span(node *yaml.Node, key string) (start, end int, found bool) {
	if node == nil {
		return 0, 0, false
	}
	at := -1
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			at = i
			break
		}
	}
	if at < 0 {
		return 0, 0, false
	}

	lines := lineOffsets(d.front)
	keyLine := node.Content[at].Line
	if keyLine < 1 || keyLine >= len(lines) {
		return 0, 0, false
	}

	last := d.endLine(lines, node.Content[at+1], node.Content[at].Column)
	if last < keyLine {
		last = keyLine
	}
	if at+2 < len(node.Content) && last > node.Content[at+2].Line-1 {
		last = node.Content[at+2].Line - 1
	}
	if last > len(lines)-1 {
		last = len(lines) - 1
	}
	return lines[keyLine-1], lines[last], true
}

// endLine is the last line of the frontmatter one node stands on. A value
// continuing past its first line stands indented past the column of the key or
// the dash it hangs from, which is the column given here.
func (d *Document) endLine(lines []int, node *yaml.Node, column int) int {
	if node == nil || node.Line < 1 {
		return 0
	}
	switch node.Kind {
	case yaml.MappingNode:
		last := node.Line
		for i := 0; i+1 < len(node.Content); i += 2 {
			if end := d.endLine(lines, node.Content[i+1], node.Content[i].Column); end > last {
				last = end
			}
		}
		return last
	case yaml.SequenceNode:
		last := node.Line
		for _, item := range node.Content {
			if end := d.endLine(lines, item, node.Column); end > last {
				last = end
			}
		}
		return last
	}

	last := node.Line
	for at := node.Line + 1; at < len(lines); at++ {
		text := string(d.front[lines[at-1]:lines[at]])
		if strings.TrimSpace(text) == "" {
			continue
		}
		if len(getIndent(text)) < column {
			break
		}
		last = at
	}
	return last
}

// readMapping is the frontmatter as YAML, or nil when there is none to read.
func (d *Document) readMapping() (*yaml.Node, error) {
	if len(bytes.TrimSpace(d.front)) == 0 {
		return nil, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(d.front, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnreadable, err)
	}
	if len(doc.Content) == 0 {
		return nil, nil
	}
	node := doc.Content[0]
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%w: it is not a mapping", ErrUnreadable)
	}
	// A key written twice names two values, and which of them the note holds is
	// not a question the block answers.
	written := make(map[string]bool, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if written[key] {
			return nil, fmt.Errorf("%w: %s is written twice", ErrUnreadable, key)
		}
		written[key] = true
	}
	return node, nil
}

// getWritableNode is the frontmatter when it is laid out so that one key can be
// changed without touching another. Reading one that is not is fine; writing to
// it is what has to be refused.
func (d *Document) getWritableNode() (*yaml.Node, error) {
	if d.isUnterminated {
		return nil, ErrUnterminated
	}
	node, err := d.readMapping()
	if err != nil || node == nil {
		return node, err
	}
	if hasAnchor(node) {
		return nil, ErrAnchored
	}
	if flow(node) {
		return nil, ErrInline
	}
	return node, nil
}

// ErrInline is what a block whose keys share their lines gets. Nothing is
// changed in it.
//
// Everything here works by replacing the lines a key occupies, and that is a
// key's own span while one line holds one key. `{title: T, id: b}` puts them
// all on one, so the span of any of them is the span of all of them. A value
// written on one line — `load: {sat: 50}` — still occupies its key's own
// lines and is replaced as it stands.
var ErrInline = errors.New("this frontmatter is written on one line, and cannot be changed a key at a time")

// ErrAnchored is a frontmatter carrying a YAML anchor. Nothing is changed in it.
//
// An anchor is read by an alias somewhere else in the block, and a splice puts
// down a value carrying no anchor. The alias then points at nothing, and the
// note stops opening at all.
var ErrAnchored = errors.New("this frontmatter carries a YAML anchor, and cannot be changed a key at a time")

// hasAnchor reports whether anything in a subtree carries an anchor.
func hasAnchor(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if hasAnchor(child) {
			return true
		}
	}
	return false
}

// ErrUnterminated is a note that opens a frontmatter block and never closes it.
// Nothing is changed in it.
var ErrUnterminated = errors.New("this note opens a frontmatter block that is never closed")

// flow reports whether a collection is written on one line, which is what puts
// two keys in one span.
func flow(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Style&yaml.FlowStyle != 0 &&
		(node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode)
}

// empty reports whether a key holds nothing at all, which is a key a first
// entry is written under. A null somebody wrote out stands on the line and is
// not one.
func empty(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null" && node.Value == ""
}

// hasFlowStyle reports whether anything in a subtree is written on one line. An
// entry of the `links:` block is replaced on its own, and that is a line at a
// time all the way down.
func hasFlowStyle(node *yaml.Node) bool {
	if flow(node) {
		return true
	}
	for _, child := range node.Content {
		if hasFlowStyle(child) {
			return true
		}
	}
	return false
}
