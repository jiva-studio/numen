package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ErrNotOurs is what changing an entry says when the entry carries something
// the application does not own. A collision is the person's win: their key is
// reported, and their link is left as they wrote it.
var ErrNotOurs = fmt.Errorf("this link carries something the application does not own")

// owned is every key an entry of the `links:` block may carry. Anything else
// in there is the person's, and the entry it sits in is left alone.
var owned = map[string]bool{
	"to": true, "role": true, "type": true, "note": true, "label": true,
}

// entry is one record of the `links:` block: what it says, and the bytes it
// occupies.
//
// The bytes are why this exists. An entry is changed by replacing its own span
// and nothing else, so the entries around it — including ones the parser could
// not act on, keys the application has never heard of, and comments somebody
// wrote to themselves — come out of a write as they went in.
type entry struct {
	link       domain.Link
	start, end int
	// readable is whether the application could act on this entry at all. One
	// it could not is still somebody's writing and is never rewritten.
	readable bool
	// ours is whether every key in it is one the application owns.
	ours bool
	// address is the node holding where the link goes, so changing it is a
	// change to that scalar. A key called `proto`, or the word `to:` inside
	// somebody's sentence, both look the same to a search and are not this.
	address *yaml.Node
}

// block is the `links:` key: where it sits, what is in it, and how it is laid
// out, so that a new entry looks like the ones already there.
type block struct {
	start, end int
	entries    []entry
	indent     string
	found      bool
}

func (d *Document) block() (block, error) {
	node, err := d.writable()
	if err != nil {
		return block{}, err
	}
	start, end, found := d.span(node, "links")
	if !found {
		return block{indent: "  "}, nil
	}
	b := block{start: start, end: end, indent: "  ", found: true}

	var items *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "links" {
			items = node.Content[i+1]
			break
		}
	}
	// An entry is replaced on its own, which is a line at a time all the way
	// down. Anything under `links:` but a block sequence, or a key holding
	// nothing, is refused and the note is left as it stands.
	switch {
	case items == nil || empty(items):
		return b, nil
	case items.Kind != yaml.SequenceNode || flowing(items):
		return block{}, ErrInline
	}

	lines := lineOffsets(d.front)
	for i, item := range items.Content {
		at := item.Line
		if at < 1 || at >= len(lines) {
			continue
		}
		last := len(lines) - 1
		if i+1 < len(items.Content) {
			last = items.Content[i+1].Line - 1
		} else if offsetLine(lines, end) > 0 {
			last = offsetLine(lines, end) - 1
		}
		for last > at {
			text := strings.TrimSpace(string(d.front[lines[last-1]:lines[last]]))
			if text != "" && !strings.HasPrefix(text, "#") {
				break
			}
			last--
		}
		b.entries = append(b.entries, entry{
			link:     linkOf(item),
			start:    lines[at-1],
			end:      lines[last],
			readable: readable(item),
			ours:     ours(item),
			address:  valueOf(item, "to"),
		})
		if i == 0 {
			b.indent = leading(string(d.front[lines[at-1]:lines[at]]))
		}
	}
	return b, nil
}

// AddLink writes one relationship into the block, leaving every other entry as
// the bytes it was.
//
// A note names another in one role: the place already named takes the role
// written, and what the person wrote on that entry is kept.
func (d *Document) AddLink(add domain.Link) error {
	b, err := d.block()
	if err != nil {
		return err
	}
	for _, e := range b.entries {
		if e.link.Target != add.Target {
			continue
		}
		if e.link.Role == add.Role {
			return nil
		}
		_, err := d.UpdateLink(add.Target, add)
		return err
	}

	rendered, err := renderEntry(add, b.indent, d.eol)
	if err != nil {
		return err
	}
	if !b.found {
		return d.set("links", append([]byte("links:"+d.eol), rendered...))
	}
	d.splice(b.end, b.end, rendered)
	return nil
}

// RemoveLink takes out every entry that goes to one place, or only the one
// carrying a role when a role is named. An empty block goes with them.
func (d *Document) RemoveLink(to domain.Address, role domain.LinkRole) (int, error) {
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	removed, kept := 0, 0
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != to || (role != "" && e.link.Role != role) {
			kept++
			continue
		}
		d.splice(e.start, e.end, nil)
		removed++
	}
	if removed > 0 && kept == 0 {
		if err := d.set("links", nil); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

// SetLinkOfType makes the block name one note under a `type`.
//
// The first entry carrying that type is pointed at the address and keeps the
// role and the words the person wrote on it; any further entry carrying it is
// taken out, so a note names one place under one type. An empty address takes
// them all out, and an empty block goes with them.
//
// An entry carrying a key the application does not own is refused. An entry the
// application cannot read carries no role, so it names nothing and is left as
// it was written.
func (d *Document) SetLinkOfType(of string, to domain.Address, role domain.LinkRole) error {
	b, err := d.block()
	if err != nil {
		return err
	}

	var carrying []int
	kept := 0
	for i, e := range b.entries {
		if e.link.Type != of || !e.readable {
			kept++
			continue
		}
		if !e.ours {
			return fmt.Errorf("%w: %s", ErrNotOurs, e.link.Target)
		}
		carrying = append(carrying, i)
	}

	for at := len(carrying) - 1; at >= 0; at-- {
		e := b.entries[carrying[at]]
		if at > 0 || to.Value == "" {
			d.splice(e.start, e.end, nil)
			continue
		}
		next := e.link
		next.Target, next.Type = to, of
		rendered, err := renderEntry(next, b.indent, d.eol)
		if err != nil {
			return err
		}
		d.splice(e.start, e.end, rendered)
	}

	switch {
	case to.Value == "":
		if len(carrying) > 0 && kept == 0 {
			return d.set("links", nil)
		}
		return nil
	case len(carrying) > 0:
		return nil
	}

	rendered, err := renderEntry(domain.Link{Target: to, Role: role, Type: of}, b.indent, d.eol)
	if err != nil {
		return err
	}
	if !b.found {
		return d.set("links", append([]byte("links:"+d.eol), rendered...))
	}
	d.splice(b.end, b.end, rendered)
	return nil
}

// UpdateLink changes what an entry says about itself without moving where it
// goes. An entry carrying anything the application does not own is refused.
func (d *Document) UpdateLink(to domain.Address, change domain.Link) (int, error) {
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	changed := 0
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != to {
			continue
		}
		if !e.ours {
			return 0, fmt.Errorf("%w: %s", ErrNotOurs, e.link.Target)
		}
		// What was not sent is kept. A caller changing a label has not asked
		// for the person's own words about why the link exists to be dropped.
		// Every field here behaves the same way, so there is one rule to hold.
		next := e.link
		if change.Role != "" {
			next.Role = change.Role
		}
		if change.Type != "" {
			next.Type = change.Type
		}
		if change.Note != "" {
			next.Note = change.Note
		}
		if change.Label != "" {
			next.Label = change.Label
		}

		rendered, err := renderEntry(next, b.indent, d.eol)
		if err != nil {
			return 0, err
		}
		d.splice(e.start, e.end, rendered)
		changed++
	}
	return changed, nil
}

// PointLinksAt sends every entry that goes to one address to another, changing
// the address and not one byte else. An entry the application could not read,
// or one carrying a key it does not own, is left alone: its link stays as it
// was written and is visible as a problem.
func (d *Document) PointLinksAt(from domain.Address, to string) (int, error) {
	if !domain.Nameable(to) {
		// Nothing a link could say reaches it. The link stays as it was
		// written, and shows as a problem.
		return 0, nil
	}
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	moved := 0
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != from || !e.readable || !e.ours {
			continue
		}
		if e.address == nil {
			continue
		}
		start, end, ok := d.scalarSpan(e.address)
		if !ok {
			continue
		}
		written, err := scalarLike(asItWasWritten(e.address.Value, to), e.address.Style)
		if err != nil {
			return moved, err
		}
		d.splice(start, end, []byte(written))
		moved++
	}
	return moved, nil
}

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
	start = lines[node.Line-1] + node.Column - 1
	if start < 0 || start >= len(d.front) {
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
		switch front[at] {
		case '\\':
			if quote == '"' {
				at++
			}
		case '\n':
			return 0, false
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

// splice puts bytes in place of a range of the frontmatter.
func (d *Document) splice(start, end int, rendered []byte) {
	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	d.front = append(front, d.front[end:]...)
}

// renderEntry writes one entry, indented as the block already is.
func renderEntry(l domain.Link, indent, eol string) ([]byte, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode([]linkEntry{{
		To:    l.Target.Written(),
		Role:  string(l.Role),
		Type:  l.Type,
		Note:  l.Note,
		Label: l.Label,
	}}); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	var block bytes.Buffer
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		block.WriteString(indent)
		block.WriteString(line)
		block.WriteString(eol)
	}
	return block.Bytes(), nil
}

// linkOf reads one entry of the block, taking what it can and leaving the rest.
func linkOf(item *yaml.Node) domain.Link {
	var l domain.Link
	if item.Kind != yaml.MappingNode {
		return l
	}
	for i := 0; i+1 < len(item.Content); i += 2 {
		value := strings.TrimSpace(item.Content[i+1].Value)
		switch item.Content[i].Value {
		case "to":
			l.Target = domain.ParseAddress(value)
		case "role":
			l.Role = domain.LinkRole(value)
		case "type":
			l.Type = value
		case "note":
			l.Note = value
		case "label":
			l.Label = value
		}
	}
	return l
}

// readable is whether the application can act on an entry at all: somewhere to
// go, and a role it has decided on.
func readable(item *yaml.Node) bool {
	l := linkOf(item)
	return item.Kind == yaml.MappingNode && l.Target.Value != "" && domain.KnownRole(l.Role)
}

// ours is whether every key in an entry is one the application owns.
func ours(item *yaml.Node) bool {
	if item.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(item.Content); i += 2 {
		if !owned[item.Content[i].Value] {
			return false
		}
	}
	return true
}

func leading(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// offsetLine is the 1-based line a byte offset begins, or zero when it is the
// end of the block.
func offsetLine(lines []int, offset int) int {
	for i, at := range lines {
		if at == offset {
			return i + 1
		}
	}
	return 0
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

// scalar is a value as YAML has to spell it, so that a name carrying `[`, `#`,
// `&` or a word YAML reads as a number goes into a file as the text it is.
// Writing it raw is how a rename makes somebody else's note unparseable.
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

// asItWasWritten is a name in the notation the name it replaces was in. Double
// brackets are how a person writes a link, and an alias after `|` or a place
// after `#` is theirs: the target is what moved, and the rest is left standing.
func asItWasWritten(was, to string) string {
	if !strings.HasPrefix(was, "[[") || !strings.HasSuffix(was, "]]") || len(was) < 4 {
		return to
	}
	return "[[" + to + keptAfterTarget(was[2:len(was)-2]) + "]]"
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

// Links are the relationships written in the `links:` block. Links written in
// prose are not among them: those are the body's.
//
// Reading does not need the block to be laid out so that one entry can be
// changed, so this asks less of it than a write does.
func (d *Document) Links() ([]domain.Link, error) {
	node, err := d.mapping()
	if err != nil || node == nil {
		return nil, err
	}
	var items *yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "links" {
			items = node.Content[i+1]
			break
		}
	}
	if items == nil || items.Kind != yaml.SequenceNode {
		return nil, nil
	}
	out := make([]domain.Link, 0, len(items.Content))
	for _, item := range items.Content {
		out = append(out, linkOf(item))
	}
	return out, nil
}
