// A note held open: how it is opened, how a new one is made, the bytes it now
// stands as, and its prose.

package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ErrUnreadable is what opening a note says when its frontmatter is not YAML.
// Such a note is never written.
var ErrUnreadable = errors.New("the frontmatter of this note cannot be read")

// ErrBodyUnwritable is a body that opens with the frontmatter delimiter. It is
// a whole note handed back as prose — a caller that read a file, changed it,
// and returned all of it. Writing it would put a second frontmatter block
// inside the first one's note, and the block that then reads as the note's own
// is the wrong one.
var ErrBodyUnwritable = errors.New(
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
	// frontmatter with a line missing. The whole of it stands as body here, so
	// nothing written to the body is written: what a write would replace is the
	// half-written block as well as the prose.
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
//
// A note whose frontmatter block is never closed holds no prose to replace —
// the whole file stands as its body — so this is ErrUnterminated and the file
// is left as it is.
func (d *Document) SetBody(body string) error {
	if d.unterminated {
		return ErrUnterminated
	}
	text := Normalised(body)
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if d.eol == "\r\n" {
		text = strings.ReplaceAll(text, "\n", "\r\n")
	}
	d.body = []byte(text)
	return nil
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
