// The frontmatter block, written one key at a time: the lines a key occupies,
// what is put in their place, and the blocks no key of which can be replaced
// alone.

package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

func render(node *yaml.Node) (string, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	// gomutants:disable-next-line BRANCH_IF reason="yaml.Encoder writing a valid node to bytes.Buffer cannot fail"
	if err := enc.Encode(node); err != nil {
		return "", err
	}
	// gomutants:disable-next-line BRANCH_IF reason="closing a yaml.Encoder backed by bytes.Buffer cannot fail"
	if err := enc.Close(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (d *Document) put(key string, value *yaml.Node) error {
	node, err := d.getWritableNode()
	// gomutants:disable-next-line BRANCH_IF reason="set re-runs getWritableNode and returns the same error before modifying the document"
	if err != nil {
		return err
	}
	name := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	pair(node, key).copyLineComment(name, value)
	rendered, err := render(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{name, value}})
	// gomutants:disable-next-line BRANCH_IF reason="render on an in-memory key-value MappingNode cannot fail"
	if err != nil {
		return err
	}
	return d.set(key, []byte(strings.ReplaceAll(rendered, "\n", d.eol)))
}

// nodePair holds the key node and the value node of one mapping key, and is
// empty when the mapping has no such key.
type nodePair struct{ name, value *yaml.Node }

func pair(node *yaml.Node, key string) nodePair {
	if node == nil || node.Kind != yaml.MappingNode {
		return nodePair{}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return nodePair{name: node.Content[i], value: node.Content[i+1]}
		}
	}
	return nodePair{}
}

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

func (p nodePair) copyLineComment(name, value *yaml.Node) {
	if p.name == nil {
		return
	}
	name.LineComment = p.name.LineComment
	if value != p.value {
		value.LineComment = p.value.LineComment
	}
}

func (d *Document) set(key string, rendered []byte) error {
	node, err := d.getWritableNode()
	if err != nil {
		return err
	}
	rendered = indentLines(rendered, d.indent(node))

	start, end, found := d.span(node, key)
	if !found {
		return d.appendKey(node, rendered)
	}

	// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="slice capacity hint does not alter the appended frontmatter bytes"
	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	return d.commit(append(front, d.front[end:]...))
}

func (d *Document) appendKey(node *yaml.Node, rendered []byte) error {
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
		d.closeDelimiter = []byte("---" + d.eol)
	}
	return nil
}

func (d *Document) commit(front []byte) error {
	was, wasMapping := d.front, d.mapping
	d.front, d.mapping = front, nil
	if _, err := d.readMapping(); err != nil {
		d.front, d.mapping = was, wasMapping
		return err
	}
	return nil
}

func (d *Document) splice(start, end int, rendered []byte) {
	// gomutants:disable-next-line ARITHMETIC_BASE,INVERT_NEGATIVES reason="slice capacity hint does not alter the appended frontmatter bytes"
	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	d.front, d.mapping = append(front, d.front[end:]...), nil
}

func (d *Document) indent(node *yaml.Node) string {
	if node == nil || len(node.Content) == 0 {
		return ""
	}
	lines := lineOffsets(d.front)
	targetLine := node.Content[0].Line
	if targetLine < 1 || targetLine >= len(lines) {
		return ""
	}
	return getIndent(string(d.front[lines[targetLine-1]:lines[targetLine]]))
}

func indentLines(rendered []byte, indent string) []byte {
	// gomutants:disable-next-line EXPRESSION_REMOVE,INVERT_LOGICAL,INTEGER_DECREMENT,BRANCH_IF reason="empty indent or empty input produces the same byte sequence through the loop"
	if indent == "" || len(rendered) == 0 {
		return rendered
	}
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="slice capacity hint does not alter the indented output bytes"
	out := make([]byte, 0, len(rendered)+4*len(indent))
	for pos := 0; pos < len(rendered); {
		end := len(rendered)
		if next := bytes.IndexByte(rendered[pos:], '\n'); next >= 0 {
			end = pos + next + 1
		}
		if len(bytes.TrimSpace(rendered[pos:end])) > 0 {
			out = append(out, indent...)
		}
		out = append(out, rendered[pos:end]...)
		pos = end
	}
	return out
}

func (d *Document) span(node *yaml.Node, key string) (start, end int, found bool) {
	if node == nil {
		return 0, 0, false
	}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="any negative sentinel triggers the keyIdx < 0 check when the key is absent"
	keyIdx := -1
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			keyIdx = i
			break
		}
	}
	if keyIdx < 0 {
		return 0, 0, false
	}

	lines := lineOffsets(d.front)
	keyLine := node.Content[keyIdx].Line
	if keyLine < 1 || keyLine >= len(lines) {
		return 0, 0, false
	}

	last := max(
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="assigning keyLine when last equals keyLine leaves last unchanged"
		d.endLine(lines, node.Content[keyIdx+1], node.Content[keyIdx].Column), keyLine)
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="clamping last when equal to the bound leaves last unchanged"
	if keyIdx+2 < len(node.Content) && last > node.Content[keyIdx+2].Line-1 {
		last = node.Content[keyIdx+2].Line - 1
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="clamping last when equal to len(lines)-1 leaves last unchanged"
	if last > len(lines)-1 {
		last = len(lines) - 1
	}
	return lines[keyLine-1], lines[last], true
}

func (d *Document) endLine(lines []int, node *yaml.Node, column int) int {
	if node == nil || node.Line < 1 {
		return 0
	}
	switch node.Kind {
	case yaml.MappingNode:
		return d.endLineOfMapping(lines, node)
	case yaml.SequenceNode:
		return d.endLineOfSequence(lines, node)
	default:
		return d.endLineOfScalar(lines, node, column)
	}
}

func (d *Document) endLineOfMapping(lines []int, node *yaml.Node) int {
	last := node.Line
	for i := 0; i+1 < len(node.Content); i += 2 {
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="updating last when end equals last leaves last unchanged"
		if end := d.endLine(lines, node.Content[i+1], node.Content[i].Column); end > last {
			last = end
		}
	}
	return last
}

func (d *Document) endLineOfSequence(lines []int, node *yaml.Node) int {
	last := node.Line
	for _, item := range node.Content {
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="updating last when end equals last leaves last unchanged"
		if end := d.endLine(lines, item, node.Column); end > last {
			last = end
		}
	}
	return last
}

func (d *Document) endLineOfScalar(lines []int, node *yaml.Node, column int) int {
	last := node.Line
	for lineIdx := node.Line + 1; lineIdx < len(lines); lineIdx++ {
		text := d.front[lines[lineIdx-1]:lines[lineIdx]]
		if len(bytes.TrimSpace(text)) == 0 {
			continue
		}
		if len(text)-len(bytes.TrimLeft(text, " \t")) < column {
			break
		}
		last = lineIdx
	}
	return last
}

func (d *Document) readMapping() (*yaml.Node, error) {
	if d.mapping != nil {
		return d.mapping, nil
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,BRANCH_IF reason="yaml.Unmarshal on whitespace-only bytes produces an empty document handled identically below"
	if len(bytes.TrimSpace(d.front)) == 0 {
		return nil, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(d.front, &doc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnreadable, err)
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
	// gomutants:disable-next-line ARITHMETIC_BASE,INTEGER_DECREMENT,INTEGER_INCREMENT reason="map capacity hint does not change duplicate key detection"
	content := make(map[string]bool, len(node.Content)/2)
	// gomutants:disable-next-line INTEGER_DECREMENT,CONDITIONALS_BOUNDARY reason="yaml.Unmarshal produces an even number of Content nodes for MappingNode"
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if content[key] {
			return nil, fmt.Errorf("%w: %s is written twice", ErrUnreadable, key)
		}
		content[key] = true
	}
	d.mapping = node
	return node, nil
}

func (d *Document) getWritableNode() (*yaml.Node, error) {
	if d.isUnterminated {
		return nil, ErrUnterminated
	}
	node, err := d.readMapping()
	// gomutants:disable-next-line EXPRESSION_REMOVE reason="hasAnchor and isFlowCollection both return false for a nil node"
	if err != nil || node == nil {
		// gomutants:disable-next-line RETURN_ZERO reason="readMapping always returns a nil node whenever err is non-nil"
		return node, err
	}
	if hasAnchor(node) {
		return nil, ErrAnchored
	}
	if isFlowCollection(node) {
		return nil, ErrInline
	}
	return node, nil
}

// ErrInline is returned for a block whose keys share lines. The block is left unchanged.
// Writes replace the lines a key occupies, which is the key's own span only while
// one line holds one key, as `{title: T, id: b}` does not.
// A value on one line, such as `load: {cpu: 50}`, is replaced as usual.
var ErrInline = errors.New("this frontmatter is written on one line, and cannot be changed a key at a time")

// ErrAnchored is a frontmatter carrying a YAML anchor. Nothing is changed in it.
//
// An anchor is read by an alias somewhere else in the block, and a splice puts
// down a value carrying no anchor. The alias then points at nothing, and the
// note stops opening at all.
var ErrAnchored = errors.New("this frontmatter has a YAML anchor, and cannot be changed a key at a time")

func hasAnchor(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return true
	}
	return slices.ContainsFunc(node.Content, hasAnchor)
}

// ErrUnterminated is a note that opens a frontmatter block and never closes it.
// Nothing is changed in it.
var ErrUnterminated = errors.New("this note opens a frontmatter block that is never closed")

func isFlowCollection(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	return node.Style&yaml.FlowStyle != 0 &&
		(node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode)
}

func isNullValue(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null" && node.Value == ""
}

func hasFlowStyle(node *yaml.Node) bool {
	if isFlowCollection(node) {
		return true
	}
	return slices.ContainsFunc(node.Content, hasFlowStyle)
}
