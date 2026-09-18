package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ErrNotOwned is what changing an entry says when the entry carries something
// the application does not own. A collision is the person's win: the link is
// named in the error and left as they wrote it.
var ErrNotOwned = fmt.Errorf("this link carries something the application does not own")

// owned is every key an entry of the `links:` block may carry. Anything else
// in there is the person's, and the entry it sits in is left alone.
var owned = map[string]bool{
	"to": true, "role": true, "type": true, "note": true, "label": true,
}

// entry is one record of the `links:` block: what it says, and the bytes it
// occupies.
//
// An entry is changed by replacing its own span and nothing else, so the
// entries around it — including ones the parser could not act on, keys the
// application has never heard of, and comments somebody wrote to themselves —
// come out of a write as they went in.
type entry struct {
	link       domain.Link
	start, end int
	// readable is whether the application could act on this entry at all. One
	// it could not is still somebody's writing and is never rewritten.
	readable bool
	// ours is whether every key in it is one the application owns.
	ours bool
	// address is the node holding where the link goes, so changing it is a
	// change to that scalar.
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
	case items.Kind != yaml.SequenceNode || hasFlowStyle(items):
		return block{}, ErrInline
	}

	lines := lineOffsets(d.front)
	for i, item := range items.Content {
		at := item.Line
		if at < 1 || at >= len(lines) {
			continue
		}
		// An entry ends at the last line of its own value. A comment or a blank
		// line below that belongs to the entry following it.
		bound := len(lines) - 1
		if i+1 < len(items.Content) {
			bound = items.Content[i+1].Line - 1
		} else if to := offsetLine(lines, end); to > 0 {
			bound = to - 1
		}
		last := d.endLine(lines, item, items.Column)
		if last > bound {
			last = bound
		}
		if last < at {
			last = at
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
			b.indent = getIndent(string(d.front[lines[at-1]:lines[at]]))
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
// An entry carrying a key the application does not own is refused, and one it
// cannot read is left as it was written.
func (d *Document) SetLinkOfType(kind string, to domain.Address, role domain.LinkRole) error {
	b, err := d.block()
	if err != nil {
		return err
	}

	var carrying []int
	kept := 0
	for i, e := range b.entries {
		if e.link.Type != kind || !e.readable {
			kept++
			continue
		}
		if !e.ours {
			return fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
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
		next.Target, next.Type = to, kind
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

	rendered, err := renderEntry(domain.Link{Target: to, Role: role, Type: kind}, b.indent, d.eol)
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
// goes. An entry carrying anything the application does not own is refused,
// and a refusal is the whole change refused.
func (d *Document) UpdateLink(to domain.Address, change domain.Link) (int, error) {
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	type edit struct {
		start, end int
		rendered   []byte
	}
	var edits []edit
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != to {
			continue
		}
		if !e.ours {
			return 0, fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
		}
		// What was not sent is kept, and every field here behaves the same way.
		next := e.link
		if change.Role != "" {
			next.Role = change.Role
		}
		if change.Type != "" {
			next.Type = change.Type
		}
		if change.Why != "" {
			next.Why = change.Why
		}
		if change.Label != "" {
			next.Label = change.Label
		}

		rendered, err := renderEntry(next, b.indent, d.eol)
		if err != nil {
			return 0, err
		}
		edits = append(edits, edit{start: e.start, end: e.end, rendered: rendered})
	}

	for _, e := range edits {
		d.splice(e.start, e.end, e.rendered)
	}
	return len(edits), nil
}

// PointLinksAt sends every entry that goes to one address to another, changing
// the address and not one byte else. An entry the application could not read,
// or one carrying a key it does not own, is left alone: its link stays as it
// was written and is visible as a problem.
func (d *Document) PointLinksAt(from domain.Address, to string) (int, error) {
	if !domain.IsNameable(to) {
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

// renderEntry writes one entry, indented as the block already is.
func renderEntry(l domain.Link, indent, eol string) ([]byte, error) {
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode([]linkEntry{{
		To:    l.Target.GetWritten(),
		Role:  string(l.Role),
		Type:  l.Type,
		Note:  l.Why,
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
			l.Why = value
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
	return item.Kind == yaml.MappingNode && l.Target.Value != "" && domain.IsKnownRole(l.Role)
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

// asItWasWritten is a name in the notation the name it replaces was in. Double
// brackets are how a person writes a link, and an alias after `|` or a place
// after `#` is theirs: the target is what moved, and the rest is left standing.
func asItWasWritten(was, to string) string {
	if !strings.HasPrefix(was, "[[") || !strings.HasSuffix(was, "]]") || len(was) < 4 {
		return to
	}
	return "[[" + to + getAfterTarget(was[2:len(was)-2]) + "]]"
}

// Links are the relationships written in the `links:` block. Links written in
// prose are not among them: those are the body's.
//
// Reading does not need the block to be laid out so that one entry can be
// changed, so this asks less of it than a write does.
func (d *Document) Links() ([]domain.Link, error) {
	node, err := d.readMapping()
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

// linkEntry is one record of the `links:` block, in the order the format
// specification lists the fields.
type linkEntry struct {
	To    string `yaml:"to"`
	Role  string `yaml:"role"`
	Type  string `yaml:"type,omitempty"`
	Note  string `yaml:"note,omitempty"`
	Label string `yaml:"label,omitempty"`
}
