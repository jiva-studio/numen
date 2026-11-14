package markdown

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNotOwned is returned when a change targets an entry that holds content the
// application does not own. On a collision the user's entry wins: the link is named
// in the error and left as written.
var ErrNotOwned = fmt.Errorf("this link has something the application does not own")

// entry is one record of the `links:` block: its content and the bytes it occupies.
//
// An entry is changed by replacing its own span only, so neighboring entries,
// including ones the parser could not interpret, unknown keys, and user comments,
// survive a write unchanged.
type entry struct {
	link       domain.Link
	start, end int
	isReadable bool
	isOwned    bool
	address    *yaml.Node
}

// block is the `links:` key: its location, its contents, and its layout, so a new
// entry matches the existing ones.
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
		itemLine := item.Line
		if itemLine < 1 || itemLine >= len(lines) {
			continue
		}
		var linked int
		if i+1 < len(items.Content) {
			linked = items.Content[i+1].Line - 1
		} else {
			linked = offsetLine(lines, end) - 1
		}
		last := min(
			// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="clamping last when equal to linked leaves last unchanged"
			d.endLine(lines, item, items.Column), linked)
		last = max(last, itemLine)
		entries = append(entries, entry{
			link:       linkOf(item),
			start:      lines[itemLine-1],
			end:        lines[last],
			isReadable: isReadable(item),
			isOwned:    hasOnlyOwnedKeys(item),
			address:    valueOf(item, "to"),
		})
		if i == 0 {
			indent = getIndent(string(d.front[lines[itemLine-1]:lines[itemLine]]))
		}
	}
	return entries, indent
}

// AddLink writes one relationship into the block and leaves every other entry
// byte-for-byte unchanged.
// An entry is addressed by its target and type; the entry of the same target and
// type takes the given role, label and marker, and keeps its reason unless one is given.
func (d *Document) AddLink(add domain.Link) error {
	b, err := d.block()
	if err != nil {
		return err
	}
	for _, e := range b.entries {
		if e.link.Target != add.Target || e.link.Type != add.Type {
			continue
		}
		next := e.link
		next.Role = add.Role
		next.Label = add.Label
		next.Marker = add.Marker
		if add.Why != "" {
			next.Why = add.Why
		}
		if next == e.link {
			return nil
		}
		rendered, err := renderEntry(next, b.indent, d.eol)
		// gomutants:disable-next-line BRANCH_IF reason="renderEntry encodes a string struct slice to bytes.Buffer and cannot fail"
		if err != nil {
			return err
		}
		d.splice(e.start, e.end, rendered)
		return nil
	}

	return d.appendEntry(b, add)
}

// SetLink makes the block hold exactly the record given for its target and type:
// the entry that goes to the target, or to any of the addresses also given, with
// the same type is replaced whole, and a block with none gains the record.
// Another entry of the same target and type is removed, so a note holds one record
// per target and type.
func (d *Document) SetLink(link domain.Link, also ...domain.Address) error {
	b, err := d.block()
	if err != nil {
		return err
	}
	found, err := findRecords(b.entries, link, also)
	if err != nil {
		return err
	}
	if len(found) == 0 {
		return d.appendEntry(b, link)
	}
	for i, e := range slices.Backward(found) {

		if i > 0 {
			d.splice(e.start, e.end, nil)
			continue
		}
		next := link
		next.Target = e.link.Target
		rendered, err := renderEntry(next, b.indent, d.eol)
		// gomutants:disable-next-line BRANCH_IF reason="renderEntry encodes a string struct slice to bytes.Buffer and cannot fail"
		if err != nil {
			return err
		}
		d.splice(e.start, e.end, rendered)
	}
	return nil
}

func findRecords(entries []entry, link domain.Link, also []domain.Address) ([]entry, error) {
	var found []entry
	for _, e := range entries {
		isTarget := e.link.Target == link.Target || slices.Contains(also, e.link.Target)
		if !isTarget || e.link.Type != link.Type {
			continue
		}
		if !e.isOwned || !e.isReadable {
			return nil, fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
		}
		found = append(found, e)
	}
	return found, nil
}

// RemoveLink takes out every entry that matches the key: the target, the type,
// and the role when one is named. An empty block goes with them.
func (d *Document) RemoveLink(key domain.LinkKey) (int, error) {
	return d.removeEntries(func(link domain.Link) bool {
		return link.Target == key.Target && link.Type == key.Type && (key.Role == "" || link.Role == key.Role)
	})
}

// RemoveTargetLinks removes every entry to the target, of every type, and only those of the role
// when one is given. A block left empty is removed too.
func (d *Document) RemoveTargetLinks(target domain.Address, role domain.LinkRole) (int, error) {
	return d.removeEntries(func(link domain.Link) bool {
		return link.Target == target && (role == "" || link.Role == role)
	})
}

func (d *Document) removeEntries(matches func(domain.Link) bool) (int, error) {
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	removed, keptCount := 0, 0
	for _, e := range slices.Backward(b.entries) {

		if !matches(e.link) {
			keptCount++
			continue
		}
		d.splice(e.start, e.end, nil)
		removed++
	}
	if removed > 0 && keptCount == 0 {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="removing links after block() validated the frontmatter always returns nil"
		if err := d.set("links", nil); err != nil {
			return removed, err
		}
	}
	return removed, nil
}

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
	for idx, c := range slices.Backward(carrying) {
		e := b.entries[c]
		if idx > 0 || to.Value == "" {
			d.splice(e.start, e.end, nil)
			continue
		}
		next := e.link
		next.Target, next.Type = to, kind
		rendered, err := renderEntry(next, b.indent, d.eol)
		// gomutants:disable-next-line BRANCH_IF reason="renderEntry encodes a string struct slice to bytes.Buffer and cannot fail"
		if err != nil {
			return err
		}
		d.splice(e.start, e.end, rendered)
	}
	return nil
}

func (d *Document) appendEntry(b block, l domain.Link) error {
	rendered, err := renderEntry(l, b.indent, d.eol)
	// gomutants:disable-next-line BRANCH_IF reason="renderEntry encodes a string struct slice to bytes.Buffer and cannot fail"
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
// The first entry with that type is pointed at the address and keeps its role and
// the user's text; further entries with that type are removed.
// An empty address removes all of them, and an empty block with them.
// An entry the parser cannot read is left as written.
func (d *Document) SetLinkOfType(kind string, to domain.Address, role domain.LinkRole) error {
	b, err := d.block()
	if err != nil {
		return err
	}

	carrying, retained, err := findTypedEntries(b.entries, kind)
	if err != nil {
		return err
	}

	// gomutants:disable-next-line BRANCH_IF reason="replaceTypedEntries only calls renderEntry and splice and cannot fail"
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
	if change.Marker != "" {
		next.Marker = change.Marker
	}
	return next
}

type linkEdit struct {
	start, end int
	rendered   []byte
}

func (d *Document) collectLinkEdits(b block, to domain.Address, change domain.Link) ([]linkEdit, error) {
	var edits []linkEdit
	for _, e := range slices.Backward(b.entries) {

		if e.link.Target != to {
			continue
		}
		if !e.isOwned {
			return nil, fmt.Errorf("%w: %s", ErrNotOwned, e.link.Target)
		}
		next := applyLinkChange(e.link, change)
		rendered, err := renderEntry(next, b.indent, d.eol)
		// gomutants:disable-next-line BRANCH_IF reason="renderEntry encodes a string struct slice to bytes.Buffer and cannot fail"
		if err != nil {
			return nil, err
		}
		edits = append(edits, linkEdit{start: e.start, end: e.end, rendered: rendered})
	}
	return edits, nil
}

// UpdateLink changes the content of an entry without changing its target
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
		// A name that cannot be a link target leaves the link unchanged; it is reported
		// as a problem.
		return 0, nil
	}
	b, err := d.block()
	if err != nil {
		return 0, err
	}

	moved := 0
	for _, e := range slices.Backward(b.entries) {

		if e.link.Target != from || !e.isReadable || !e.isOwned {
			continue
		}
		// gomutants:disable-next-line BRANCH_IF reason="an entry with isReadable true always has a non-nil to address node"
		if e.address == nil {
			continue
		}
		start, end, ok := d.scalarSpan(e.address)
		if !ok {
			continue
		}
		content, err := scalarLike(keepLinkNotation(e.address.Value, to), e.address.Style)
		// gomutants:disable-next-line BRANCH_IF reason="scalarLike encodes a scalar string to bytes.Buffer and cannot fail"
		if err != nil {
			return moved, err
		}
		d.splice(start, end, []byte(content))
		moved++
	}
	return moved, nil
}
