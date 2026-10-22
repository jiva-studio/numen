package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ErrNotOwned is what changing an entry says when the entry carries something
// the application does not own. A collision is the person's win: the link is
// named in the error and left as they wrote it.
var ErrNotOwned = fmt.Errorf("this link carries something the application does not own")

// owned is every key an entry of the `links:` block may carry. Anything else
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
	// isReadable is whether the application could act on this entry at all. One
	// it could not is still somebody's writing and is never rewritten.
	isReadable bool
	// isOwned is whether every key in it is one the application owns.
	isOwned bool
	// address is the node holding where the link goes, so changing it is a
	// change to that scalar.
	address *yaml.Node
}

// block is the `links:` key: where it is located, what is in it, and how it is laid
// out, so that a new entry looks like the ones already there.
type block struct {
	start, end int
	entries    []entry
	indent     string
	isFound    bool
}

func (d *Document) block() (block, error) {
	node, err := d.getWritableNode()
	if err != nil {
		return block{}, err
	}
	start, end, found := d.span(node, "links")
	if !found {
		return block{indent: "  "}, nil
	}
	b := block{start: start, end: end, indent: "  ", isFound: true}

	items := valueOf(node, "links")
	switch {
	case items == nil || isNullValue(items):
		return b, nil
	case items.Kind != yaml.SequenceNode || hasFlowStyle(items):
		return block{}, ErrInline
	}

	entries, indent := d.readBlockEntries(items, end)
	b.entries = entries
	if len(entries) > 0 {
		b.indent = indent
	}
	return b, nil
}

func (d *Document) readBlockEntries(items *yaml.Node, end int) ([]entry, string) {
	lines := lineOffsets(d.front)
	var entries []entry
	var indent string
	for i, item := range items.Content {
		at := item.Line
		if at < 1 || at >= len(lines) {
			continue
		}
		var linked int
		if i+1 < len(items.Content) {
			linked = items.Content[i+1].Line - 1
		} else {
			linked = offsetLine(lines, end) - 1
		}
		last := d.endLine(lines, item, items.Column)
		if last > linked {
			last = linked
		}
		if last < at {
			last = at
		}
		entries = append(entries, entry{
			link:       linkOf(item),
			start:      lines[at-1],
			end:        lines[last],
			isReadable: isReadable(item),
			isOwned:    hasOnlyOwnedKeys(item),
			address:    valueOf(item, "to"),
		})
		if i == 0 {
			indent = getIndent(string(d.front[lines[at-1]:lines[at]]))
		}
	}
	return entries, indent
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
		if e.link.Role == add.Role && e.link.Label == add.Label && e.link.Type == add.Type {
			return nil
		}
		next := e.link
		next.Role = add.Role
		next.Label = add.Label
		if add.Type != "" {
			next.Type = add.Type
		}
		rendered, err := renderEntry(next, b.indent, d.eol)
		if err != nil {
			return err
		}
		d.splice(e.start, e.end, rendered)
		return nil
	}

	rendered, err := renderEntry(add, b.indent, d.eol)
	if err != nil {
		return err
	}
	if !b.isFound {
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

	removed, keptCount := 0, 0
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != to || (role != "" && e.link.Role != role) {
			keptCount++
			continue
		}
		d.splice(e.start, e.end, nil)
		removed++
	}
	if removed > 0 && keptCount == 0 {
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
func findTypedEntries(entries []entry, kind string) ([]int, int, error) {
	var carrying []int
	remainingCount := 0
	for i, e := range entries {
		if e.link.Type != kind || !e.isReadable {
			remainingCount++
			continue
		}
		if !e.isOwned {
			return nil, 0, fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
		}
		carrying = append(carrying, i)
	}
	return carrying, remainingCount, nil
}

func (d *Document) replaceTypedEntries(b block, carrying []int, kind string, to domain.Address) error {
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
	return nil
}

func (d *Document) appendEntry(b block, l domain.Link) error {
	rendered, err := renderEntry(l, b.indent, d.eol)
	if err != nil {
		return err
	}
	if !b.isFound {
		return d.set("links", append([]byte("links:"+d.eol), rendered...))
	}
	d.splice(b.end, b.end, rendered)
	return nil
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

	carrying, retained, err := findTypedEntries(b.entries, kind)
	if err != nil {
		return err
	}

	if err := d.replaceTypedEntries(b, carrying, kind, to); err != nil {
		return err
	}

	if to.Value == "" {
		if len(carrying) > 0 && retained == 0 {
			return d.set("links", nil)
		}
		return nil
	}
	if len(carrying) > 0 {
		return nil
	}
	return d.appendEntry(b, domain.Link{Target: to, Role: role, Type: kind})
}

func applyLinkChange(orig, change domain.Link) domain.Link {
	next := orig
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
	return next
}

type linkEdit struct {
	start, end int
	rendered   []byte
}

func (d *Document) collectLinkEdits(b block, to domain.Address, change domain.Link) ([]linkEdit, error) {
	var edits []linkEdit
	for i := len(b.entries) - 1; i >= 0; i-- {
		e := b.entries[i]
		if e.link.Target != to {
			continue
		}
		if !e.isOwned {
			return nil, fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
		}
		next := applyLinkChange(e.link, change)
		rendered, err := renderEntry(next, b.indent, d.eol)
		if err != nil {
			return nil, err
		}
		edits = append(edits, linkEdit{start: e.start, end: e.end, rendered: rendered})
	}
	return edits, nil
}

// UpdateLink changes what an entry says about itself without moving where it
// goes. An entry carrying anything the application does not own is refused,
// and a refusal is the whole change refused.
func (d *Document) UpdateLink(to domain.Address, change domain.Link) (int, error) {
	b, err := d.block()
	if err != nil {
		return 0, err
	}
	edits, err := d.collectLinkEdits(b, to, change)
	if err != nil {
		return 0, err
	}
	for _, e := range edits {
		d.splice(e.start, e.end, e.rendered)
	}
	return len(edits), nil
}

// RedirectLinks sends every entry that goes to one address to another, changing
// the address and not one byte else. An entry the application could not read,
// or one carrying a key it does not own, is left alone: its link stays as it
// was written and is visible as a problem.
func (d *Document) RedirectLinks(from domain.Address, to string) (int, error) {
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
		if e.link.Target != from || !e.isReadable || !e.isOwned {
			continue
		}
		if e.address == nil {
			continue
		}
		start, end, ok := d.scalarSpan(e.address)
		if !ok {
			continue
		}
		content, err := scalarLike(keepLinkNotation(e.address.Value, to), e.address.Style)
		if err != nil {
			return moved, err
		}
		d.splice(start, end, []byte(content))
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

	text := bytes.TrimRight(out.Bytes(), "\n")
	block := make([]byte, 0, len(text)+(bytes.Count(text, []byte("\n"))+1)*(len(indent)+len(eol)))
	for {
		line, rest, more := bytes.Cut(text, []byte("\n"))
		block = append(block, indent...)
		block = append(block, line...)
		block = append(block, eol...)
		if !more {
			return block, nil
		}
		text = rest
	}
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

// isReadable is whether the application can act on an entry at all: somewhere to
// go, and a role it has decided on.
func isReadable(item *yaml.Node) bool {
	if item.Kind != yaml.MappingNode {
		return false
	}
	l := linkOf(item)
	return l.Target.Value != "" && domain.IsKnownRole(l.Role)
}

// hasOnlyOwnedKeys is whether every key in an entry is one the application owns.
func hasOnlyOwnedKeys(item *yaml.Node) bool {
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

// keepLinkNotation is a name in the notation the name it replaces was in. Double
// brackets are how a person writes a link, and an alias after `|` or a place
// after `#` is theirs: the target is what moved, and the rest is left standing.
func keepLinkNotation(was, to string) string {
	if !strings.HasPrefix(was, "[[") || !strings.HasSuffix(was, "]]") || len(was) < 4 {
		return to
	}
	return "[[" + to + getWikilinkSuffix(was[2:len(was)-2]) + "]]"
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
