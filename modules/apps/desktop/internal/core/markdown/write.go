package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// ErrUnreadable is what opening a note says when its frontmatter is not YAML.
// Such a note is never written: repairing the block means guessing at what the
// person wrote, and rewriting around it means dropping what could not be read.
var ErrUnreadable = errors.New("the frontmatter of this note cannot be read")

// Document is a note held open so that one part of it can be changed and every
// other part left as the bytes it arrived as.
//
// This is why it is not a struct marshalled back out. The frontmatter is shared
// with the person: their key order, their comments, their quoting
// and their line endings are theirs, and a writer that rebuilds the block from
// what it understands returns a file full of changes nobody asked for. So a
// change here is a splice — the span of one key is replaced, and the rest of
// the file is never rewritten at all.
type Document struct {
	bom   []byte
	open  []byte // the opening `---` line, with its ending
	front []byte // between the delimiters, each line with its own ending
	shut  []byte // the closing `---` line, with its ending
	body  []byte
	eol   string
	// unterminated is a file that opens with the delimiter and never closes it.
	unterminated bool
}

// Open reads a note for changing.
func Open(raw []byte) (*Document, error) {
	d := &Document{eol: lineEnding(raw)}

	rest := raw
	if after, found := bytes.CutPrefix(rest, []byte("\xef\xbb\xbf")); found {
		d.bom, rest = []byte("\xef\xbb\xbf"), after
	}

	first, _, found := bytes.Cut(rest, []byte("\n"))
	if !found || strings.TrimRight(string(first), "\r") != "---" {
		d.body = rest
		return d, nil
	}

	from := len(first) + 1
	for at := from; at < len(rest); {
		line, next := rest[at:], len(rest)
		if end := bytes.IndexByte(rest[at:], '\n'); end >= 0 {
			line, next = rest[at:at+end], at+end+1
		}
		if strings.TrimRight(string(line), "\r") == "---" {
			d.open = rest[:from]
			d.front = rest[from:at]
			d.shut = rest[at:next]
			d.body = rest[next:]
			if _, err := d.mapping(); err != nil {
				return nil, err
			}
			return d, nil
		}
		at = next
	}

	// Unterminated: not a frontmatter block, and the whole file is body — but a
	// file that opens with the delimiter and never closes it is somebody's
	// frontmatter with a line missing, not prose that happens to start that way.
	// Writing would put a second block above the first and turn their keys into
	// text, so it is refused instead.
	d.body = rest
	d.unterminated = true
	return d, nil
}

// Create is a new note: an identifier, because the application is making this
// one, and whatever the person is starting it with.
func Create(identifier, body string) []byte {
	var out bytes.Buffer
	out.WriteString("---\nid: ")
	out.WriteString(identifier)
	out.WriteString("\n---\n")
	out.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		out.WriteString("\n")
	}
	return out.Bytes()
}

// Bytes is the note as it now stands.
func (d *Document) Bytes() []byte {
	out := make([]byte, 0, len(d.bom)+len(d.open)+len(d.front)+len(d.shut)+len(d.body))
	out = append(out, d.bom...)
	out = append(out, d.open...)
	out = append(out, d.front...)
	out = append(out, d.shut...)
	return append(out, d.body...)
}

// Body is the prose below the frontmatter.
func (d *Document) Body() string { return string(d.body) }

// SetBody replaces the prose and leaves the frontmatter alone. The prose is
// written with the file's own line ending, and ends with one.
func (d *Document) SetBody(body string) {
	text := Normalised(body)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if d.eol == "\r\n" {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	d.body = []byte(text)
}

// Identifier is what the note carries, and whether it carries one.
func (d *Document) Identifier() (string, bool) {
	node, err := d.mapping()
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
	return d.set("id", []byte("id: "+identifier+d.eol))
}

// Title is what the frontmatter says the note is called, and whether it says.
// A key with nothing in it names nothing, and the note is named by what comes
// after it.
func (d *Document) Title() (string, bool) {
	node, err := d.mapping()
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
	written, err := scalar(title)
	if err != nil {
		return err
	}
	return d.set("title", []byte("title: "+strings.ReplaceAll(written, "\n", d.eol)+d.eol))
}

// set replaces the lines one top-level key occupies, or appends them when the
// key is not there yet. Empty replacement removes the key.
func (d *Document) set(key string, rendered []byte) error {
	node, err := d.writable()
	if err != nil {
		return err
	}

	start, end, found := d.span(node, key)
	if !found {
		if len(rendered) == 0 {
			return nil
		}
		if node == nil && len(d.open) == 0 {
			// A note with no frontmatter grows one.
			d.open = []byte("---" + d.eol)
			d.shut = []byte("---" + d.eol)
		}
		if len(d.front) > 0 && !bytes.HasSuffix(d.front, []byte("\n")) {
			d.front = append(d.front, []byte(d.eol)...)
		}
		d.front = append(append([]byte(nil), d.front...), rendered...)
		return nil
	}

	front := make([]byte, 0, len(d.front)-(end-start)+len(rendered))
	front = append(front, d.front[:start]...)
	front = append(front, rendered...)
	d.front = append(front, d.front[end:]...)
	return nil
}

// span is the byte range one top-level key occupies in the frontmatter,
// including the lines its value continues onto.
//
// The end is walked back over blank lines and comments, because a comment
// written above the next key belongs to that key and not to this one. Taking
// it with the entry being replaced would delete somebody's note to themselves
// on the way past.
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

	last := len(lines) - 1
	if at+2 < len(node.Content) {
		last = node.Content[at+2].Line - 1
	}
	for last > keyLine {
		text := strings.TrimSpace(string(d.front[lines[last-1]:lines[last]]))
		if text != "" && !strings.HasPrefix(text, "#") {
			break
		}
		last--
	}
	return lines[keyLine-1], lines[last], true
}

// mapping is the frontmatter as YAML, or nil when there is none to read.
func (d *Document) mapping() (*yaml.Node, error) {
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
	return node, nil
}

// writable is the frontmatter when it is laid out so that one key can be
// changed without touching another. Reading one that is not is fine; writing to
// it is what has to be refused.
func (d *Document) writable() (*yaml.Node, error) {
	if d.unterminated {
		return nil, ErrUnterminated
	}
	node, err := d.mapping()
	if err != nil || node == nil {
		return node, err
	}
	if err := inline(node); err != nil {
		return nil, err
	}
	return node, nil
}

// ErrInline is what a frontmatter block written on one line gets. Nothing is
// changed in it.
//
// Everything here works by replacing the lines a key occupies, and that is only
// a key's own span while one line holds one key. `{title: T, id: b}` puts them
// all on one, so the span of any of them is the span of all of them, and a
// write meant for one would take the rest with it. Refusing is the only honest
// answer: the alternative is to reformat somebody's file to suit the writer.
var ErrInline = errors.New("this frontmatter is written on one line, and cannot be changed a key at a time")

// ErrUnterminated is a note that opens a frontmatter block and never closes it.
// What the person meant is not knowable from here, and writing would decide it
// for them.
var ErrUnterminated = errors.New("this note opens a frontmatter block that is never closed")

// inline refuses a block whose layout the splice cannot reason about. It looks
// through the whole tree, because a flow sequence for `links:` breaks the same
// arithmetic one level down.
func inline(node *yaml.Node) error {
	if node.Style&yaml.FlowStyle != 0 && (node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode) {
		return ErrInline
	}
	for _, child := range node.Content {
		if err := inline(child); err != nil {
			return err
		}
	}
	return nil
}

// lineOffsets is where each line of a block begins, with the end of the block
// as a final entry, so that line n runs from offsets[n-1] to offsets[n].
func lineOffsets(block []byte) []int {
	offsets := []int{0}
	for i, b := range block {
		if b == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	if offsets[len(offsets)-1] != len(block) {
		offsets = append(offsets, len(block))
	}
	return offsets
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

// asWritten is an address as it goes into a file. A name is written as itself:
// `name://` is how the index holds it and never appears in a note.
func asWritten(a domain.Address) string {
	if a.Scheme == domain.SchemeName {
		return a.Value
	}
	return a.String()
}
