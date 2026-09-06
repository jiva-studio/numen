package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// ErrUnreadable is what opening a note says when its frontmatter is not YAML.
// Such a note is never written.
var ErrUnreadable = errors.New("the frontmatter of this note cannot be read")

// ErrBodyRefused is a body that opens with the frontmatter delimiter. It is a
// whole note handed back as prose — a caller that read a file, changed it, and
// returned all of it. Writing it would put a second frontmatter block inside
// the first one's note, and the block that then reads as the note's own is the
// wrong one.
var ErrBodyRefused = errors.New(
	"a body is the prose below the frontmatter, and this one begins with a frontmatter block; " +
		"send what note_read gave you, or use the link tools to change the frontmatter")

// OpensFrontmatter reports whether text begins a frontmatter block. A byte
// order mark stands before the delimiter and is no part of the prose.
func OpensFrontmatter(text string) bool {
	opening := strings.TrimPrefix(text, "\ufeff")
	return strings.HasPrefix(opening, "---\n") || strings.HasPrefix(opening, "---\r\n")
}

// Document is a note held open so that one part of it can be changed and every
// other part left as the bytes it arrived as.
//
// The frontmatter is shared with the person: their key order, their comments,
// their quoting and their line endings are theirs. A change here is a splice —
// the span of one key is replaced, and the rest of the file is never rewritten.
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

// OpenBody holds prose alone open for changing. There is no frontmatter to
// read, so the whole of it is body, and the line ending is the prose's own.
func OpenBody(body []byte) *Document {
	return &Document{eol: lineEnding(body), body: body}
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

	// A file that opens with the delimiter and never closes it is somebody's
	// frontmatter with a line missing. The whole of it stands as body here.
	// The frontmatter writers and SpliceBody refuse it; SetBody and
	// PointProseAt do not, and either replaces the file, half-written block and
	// all.
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

// SpliceBody replaces one run of the prose, addressed by bytes of the body as
// Body hands it over. Every byte outside the run is left as it arrived, and
// what is written in its place takes the file's own line ending.
func (d *Document) SpliceBody(start, end int, text string) error {
	if d.unterminated {
		return ErrUnterminated
	}
	if start < 0 || end < start || end > len(d.body) {
		return fmt.Errorf("splice %d:%d is outside a body of %d bytes", start, end, len(d.body))
	}

	written := Normalised(text)
	if d.eol == "\r\n" {
		written = strings.ReplaceAll(written, "\n", "\r\n")
	}

	body := make([]byte, 0, len(d.body)-(end-start)+len(written))
	body = append(body, d.body[:start]...)
	body = append(body, written...)
	d.body = append(body, d.body[end:]...)
	return nil
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
	return d.put("id", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: identifier})
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
	node, err := d.mapping()
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

	spelled := d.spelling(key)
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
	// Verbatim keeps what the entry holds as it was written, which is what an
	// entry the application could not read gets. Such an entry under a key
	// that is not there is written nowhere.
	Verbatim bool
}

// EntryNames is the keys of the mapping under one top-level frontmatter key, in
// the order it holds them. A key that is not there holds none.
//
// False is a key holding something other than a mapping. Such a value is the
// person's whole, and the entries of a mapping are what this writes.
func (d *Document) EntryNames(key string) ([]string, bool) {
	node, err := d.mapping()
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

	node, err := d.mapping()
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
		if !one.Verbatim {
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

// spelling is how each name of one key's list is quoted, so that a name coming
// through a write untouched comes through spelled as it was.
func (d *Document) spelling(key string) map[string]yaml.Style {
	out := map[string]yaml.Style{}
	node, err := d.mapping()
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
	node, err := d.writable()
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
	node, err := d.writable()
	if err != nil {
		return err
	}
	rendered = indented(rendered, d.indent(node))

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
	if _, err := d.mapping(); err != nil {
		d.front = was
		return err
	}
	return nil
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
	return leading(string(d.front[lines[at-1]:lines[at]]))
}

// indented is rendered lines written at the indentation the block's keys stand
// at. A line with nothing on it takes none.
func indented(rendered []byte, indent string) []byte {
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
		if len(leading(text)) < column {
			break
		}
		last = at
	}
	return last
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
	if anchored(node) {
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

// anchored reports whether anything in a subtree carries an anchor.
func anchored(node *yaml.Node) bool {
	if node == nil {
		return false
	}
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return true
	}
	for _, child := range node.Content {
		if anchored(child) {
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

// flowing reports whether anything in a subtree is written on one line. An
// entry of the `links:` block is replaced on its own, and that is a line at a
// time all the way down.
func flowing(node *yaml.Node) bool {
	if flow(node) {
		return true
	}
	for _, child := range node.Content {
		if flowing(child) {
			return true
		}
	}
	return false
}

// breakWidth is the bytes the line break standing at one offset takes, and zero
// where no break stands there. YAML breaks a line on a carriage return, a line
// feed, the two together, and the Unicode NEL, LS and PS.
func breakWidth(block []byte, at int) int {
	switch block[at] {
	case '\r':
		if at+1 < len(block) && block[at+1] == '\n' {
			return 2
		}
		return 1
	case '\n':
		return 1
	case 0xC2:
		if at+1 < len(block) && block[at+1] == 0x85 {
			return 2
		}
	case 0xE2:
		if at+2 < len(block) && block[at+1] == 0x80 && (block[at+2] == 0xA8 || block[at+2] == 0xA9) {
			return 3
		}
	}
	return 0
}

// endsWithBreak reports whether a block's last line is finished.
func endsWithBreak(block []byte) bool {
	for width := 1; width <= 3 && width <= len(block); width++ {
		if breakWidth(block, len(block)-width) == width {
			return true
		}
	}
	return false
}

// lineOffsets is where each line of a block begins, with the end of the block
// as a final entry, so that line n runs from offsets[n-1] to offsets[n].
func lineOffsets(block []byte) []int {
	offsets := []int{0}
	for at := 0; at < len(block); {
		width := breakWidth(block, at)
		if width == 0 {
			at++
			continue
		}
		at += width
		offsets = append(offsets, at)
	}
	if offsets[len(offsets)-1] != len(block) {
		offsets = append(offsets, len(block))
	}
	return offsets
}

// columnOffset is where the column the parser reports on one line stands in the
// block, in bytes. The parser counts a column in characters, and a column past
// the end of its line stands at the end of it.
func columnOffset(block []byte, lines []int, line, column int) int {
	at, end := lines[line-1], lines[line]
	for count := 1; count < column && at < end; count++ {
		_, width := utf8.DecodeRune(block[at:end])
		at += width
	}
	return at
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
