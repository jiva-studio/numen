// The keys the application owns in a note, one at a time: what each holds and
// what it is written as.

package markdown

import (
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Identifier is what the note carries, and whether it carries one.
func (d *Document) Identifier() (string, bool) {
	node, err := d.readMapping()
	if err != nil || node == nil {
		return "", false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "id" {
			return strings.TrimSpace(node.Content[i+1].Value), true
		}
	}
	return "", false
}

// SetIdentifier writes the identifier the note is to carry from now on.
func (d *Document) SetIdentifier(identifier string) error {
	return d.put("id", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: identifier})
}

// Title is what the frontmatter says the note is called, and whether it says.
// A key with nothing in it names nothing, and the note is named by what comes
// after it.
func (d *Document) Title() (string, bool) {
	node, err := d.readMapping()
	if err != nil || node == nil {
		return "", false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == "title" {
			title := strings.TrimSpace(node.Content[i+1].Value)
			return title, title != ""
		}
	}
	return "", false
}

// SetTitle writes the title the note is shown by from now on.
func (d *Document) SetTitle(title string) error {
	var held yaml.Node
	if err := held.Encode(title); err != nil {
		return err
	}
	return d.put("title", &held)
}

// List is the names one top-level frontmatter key holds, in the order they
// stand in it. False when the key is not there, and when it holds anything but
// a list. An entry that is not text is not a name and is not among them.
func (d *Document) List(key string) ([]string, bool) {
	node, err := d.readMapping()
	if err != nil || node == nil {
		return nil, false
	}
	items := valueOf(node, key)
	if items == nil || items.Kind != yaml.SequenceNode {
		return nil, false
	}
	out := make([]string, 0, len(items.Content))
	for _, item := range items.Content {
		if item.Kind == yaml.ScalarNode && item.Tag == "!!str" {
			out = append(out, item.Value)
		}
	}
	return out, true
}

// SetList writes the names one top-level frontmatter key holds from now on,
// one to a line. No names removes the key.
//
// A name already in the list is written the way it was written.
func (d *Document) SetList(key string, names []string) error {
	if len(names) == 0 {
		return d.set(key, nil)
	}

	spelled := d.getSpelling(key)
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, name := range names {
		seq.Content = append(seq.Content, &yaml.Node{
			Kind: yaml.ScalarNode, Style: spelled[name], Value: name,
		})
	}
	return d.put(key, seq)
}

// Entry is one line of a mapping written under a top-level frontmatter key.
type Entry struct {
	Key   string
	Value any
	// IsVerbatim keeps what the entry holds as it was written, which is what an
	// entry the application could not read gets. Such an entry under a key
	// that is not there is written nowhere.
	IsVerbatim bool
}

// EntryNames is the keys of the mapping under one top-level frontmatter key, in
// the order it holds them. A key that is not there holds none.
//
// False is a key holding something other than a mapping. Such a value is the
// person's whole, and the entries of a mapping are what this writes.
func (d *Document) EntryNames(key string) ([]string, bool) {
	node, err := d.readMapping()
	if err != nil {
		return nil, false
	}
	held := d.entries(node, key)
	if held == nil {
		return nil, false
	}
	out := make([]string, 0, len(held.Content)/2)
	for i := 0; i+1 < len(held.Content); i += 2 {
		out = append(out, held.Content[i].Value)
	}
	return out, true
}

// entries is the mapping one top-level key holds, an empty one where the key
// holds nothing at all or is not there, and nil where it holds something else.
func (d *Document) entries(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return &yaml.Node{Kind: yaml.MappingNode}
	}
	held := valueOf(node, key)
	switch {
	case held == nil || empty(held):
		return &yaml.Node{Kind: yaml.MappingNode}
	case held.Kind != yaml.MappingNode:
		return nil
	}
	return held
}

// SetMapping writes the entries one top-level frontmatter key holds from now
// on, one to a line and in the order they are given. No entries removes the
// key.
func (d *Document) SetMapping(key string, entries []Entry) error {
	if len(entries) == 0 {
		return d.set(key, nil)
	}

	node, err := d.readMapping()
	if err != nil {
		return err
	}
	existing := d.entries(node, key)
	if existing == nil {
		existing = &yaml.Node{Kind: yaml.MappingNode}
	}
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	for _, one := range entries {
		was := pair(existing, one.Key)
		held := was.value
		if !one.IsVerbatim {
			held = &yaml.Node{}
			if err := held.Encode(one.Value); err != nil {
				return err
			}
		}
		if held == nil {
			continue
		}
		name := &yaml.Node{Kind: yaml.ScalarNode, Value: one.Key}
		was.carry(name, held)
		mapping.Content = append(mapping.Content, name, held)
	}
	if len(mapping.Content) == 0 {
		return d.set(key, nil)
	}
	return d.put(key, mapping)
}

// Scalar is what one top-level frontmatter key holds, and whether it holds
// anything. A key holding a list or a mapping holds no scalar.
func (d *Document) Scalar(key string) (string, bool) {
	node, err := d.readMapping()
	if err != nil || node == nil {
		return "", false
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != key {
			continue
		}
		if node.Content[i+1].Kind != yaml.ScalarNode {
			return "", false
		}
		held := strings.TrimSpace(node.Content[i+1].Value)
		return held, held != ""
	}
	return "", false
}

// SetScalar writes what one top-level frontmatter key holds from now on. An
// empty value removes the key.
func (d *Document) SetScalar(key, value string) error {
	if value == "" {
		return d.set(key, nil)
	}
	return d.SetValue(key, value)
}

// SetValue writes what one top-level frontmatter key holds from now on, in the
// spelling its own type is written in: a number stands as a number, and true
// and false stand as themselves.
func (d *Document) SetValue(key string, value any) error {
	var held yaml.Node
	if err := held.Encode(value); err != nil {
		return err
	}
	return d.put(key, &held)
}

// SetDay writes the day one top-level frontmatter key stands for from now on,
// as a day with no hour on it.
func (d *Document) SetDay(key string, day time.Time) error {
	return d.put(key, &yaml.Node{
		Kind: yaml.ScalarNode, Tag: "!!timestamp", Value: day.Format("2006-01-02"),
	})
}

// getSpelling is how each name of one key's list is quoted, so that a name
// coming through a write untouched comes through spelled as it was.
func (d *Document) getSpelling(key string) map[string]yaml.Style {
	out := map[string]yaml.Style{}
	node, err := d.readMapping()
	if err != nil || node == nil {
		return out
	}
	items := valueOf(node, key)
	if items == nil || items.Kind != yaml.SequenceNode {
		return out
	}
	for _, item := range items.Content {
		if item.Kind == yaml.ScalarNode {
			out[item.Value] = item.Style
		}
	}
	return out
}
