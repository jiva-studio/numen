package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrUnreadable is returned when opening a note whose frontmatter is not YAML.
// Such a note is never written.
var ErrUnreadable = errors.New("the frontmatter of this note cannot be read")

// ErrBodyUnwritable is a body that opens with the frontmatter delimiter. It is
// a whole note passed as prose, for example by a caller that read a file,
// changed it, and returned all of it. Writing it would nest a second
// frontmatter block in the note, and the block parsed as the note's frontmatter
// would be the wrong one.
var ErrBodyUnwritable = errors.New(
	"a body is the prose below the frontmatter, and this one begins with a frontmatter block; " +
		"send what note_read gave you, or use the link tools to change the frontmatter")

// IsFrontmatterStart reports whether text begins a frontmatter block. A byte
// order mark before the delimiter is ignored and is not part of the prose.
func IsFrontmatterStart(text string) bool {
	opening := strings.TrimPrefix(text, "\ufeff")
	return strings.HasPrefix(opening, "---\n") || strings.HasPrefix(opening, "---\r\n")
}

// other part left as the bytes it arrived as.
//
// The frontmatter is user-authored: key order, comments, quoting and line
// endings are preserved. A change is a splice: the span of one key is
// replaced, and the rest of the file is never rewritten.
type Document struct {
	bom            []byte
	open           []byte // the opening `---` line, with its ending
	front          []byte // between the delimiters, each line with its own ending
	closeDelimiter []byte // the closing `---` line, with its ending
	body           []byte
	eol            string
	// mapping is the frontmatter parsed, held until the frontmatter changes.
	mapping *yaml.Node
	// isUnterminated is a file that opens with the delimiter and never closes it.
	isUnterminated bool
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
	for offset := from; offset < len(rest); {
		line, next := rest[offset:], len(rest)
		if end := bytes.IndexByte(rest[offset:], '\n'); end >= 0 {
			line, next = rest[offset:offset+end], offset+end+1
		}
		if strings.TrimRight(string(line), "\r") == "---" {
			d.open = rest[:from]
			d.front = rest[from:offset]
			d.closeDelimiter = rest[offset:next]
			d.body = rest[next:]
			if _, err := d.readMapping(); err != nil {
				return nil, err
			}
			return d, nil
		}
		offset = next
	}

	// A file that opens with the delimiter and never closes it has an incomplete
	// frontmatter block. The whole file is kept as the body, and body writes are
	// rejected, because a write would replace the incomplete block together with
	// the prose.
	d.body = rest
	d.isUnterminated = true
	return d, nil
}

// Create returns the bytes of a new note: frontmatter with the given
// identifier, followed by the initial body.
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

// Bytes returns the current content of the note.
func (d *Document) Bytes() []byte {
	out := make([]byte, 0, len(d.bom)+len(d.open)+len(d.front)+len(d.closeDelimiter)+len(d.body))
	out = append(out, d.bom...)
	out = append(out, d.open...)
	out = append(out, d.front...)
	out = append(out, d.closeDelimiter...)
	return append(out, d.body...)
}

// Body is the prose below the frontmatter.
func (d *Document) Body() string { return string(d.body) }

// SetBody replaces the prose and leaves the frontmatter alone. The prose is
// written with the file's own line ending, and ends with one.
//
// A note whose frontmatter block is never closed holds no prose to replace —
// is left as it is.
func (d *Document) SetBody(body string) error {
	if d.isUnterminated {
		return ErrUnterminated
	}
	text := Normalise(body)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if d.eol == "\r\n" {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	d.body = []byte(text)
	return nil
}

// SpliceBody replaces one run of the prose, addressed by byte offsets into the
// body as returned by Body. Every byte outside the run is unchanged, and the
// replacement text uses the file's own line ending.
func (d *Document) SpliceBody(start, end int, text string) error {
	if d.isUnterminated {
		return ErrUnterminated
	}
	if start < 0 || end < start || end > len(d.body) {
		return fmt.Errorf("splice %d:%d is outside a body of %d bytes", start, end, len(d.body))
	}

	content := Normalise(text)
	if d.eol == "\r\n" {
		content = strings.ReplaceAll(content, "\n", "\r\n")
	}

	body := make([]byte, 0, len(d.body)-(end-start)+len(content))
	body = append(body, d.body[:start]...)
	body = append(body, content...)
	d.body = append(body, d.body[end:]...)
	return nil
}
