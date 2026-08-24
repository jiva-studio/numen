package markdown

import (
	"bytes"
	"errors"
)

// ErrNotAHeading is a text a level-one heading is read back as something else.
// A heading is one line, and a run of hashes at the end of one closes it.
var ErrNotAHeading = errors.New("a level-one heading cannot say this")

// Headable reports whether a level-one heading written with this text is read
// back as the text.
func Headable(text string) bool {
	found := headings([]byte("# " + text))
	return len(found) == 1 && found[0].Level == 1 && found[0].Text == text
}

// SetHeading writes the text of the note's first level-one heading, and says
// whether the note has one. Only that line is replaced.
func (d *Document) SetHeading(text string) (bool, error) {
	for _, h := range headings(d.body) {
		if h.Level != 1 {
			continue
		}
		if !Headable(text) {
			return false, ErrNotAHeading
		}
		end := lineEnd(d.body, h.Offset)
		body := make([]byte, 0, len(d.body)-(end-h.Offset)+len(text)+2)
		body = append(body, d.body[:h.Offset]...)
		body = append(body, "# "+text...)
		d.body = append(body, d.body[end:]...)
		return true, nil
	}
	return false, nil
}

// InsertHeading opens the prose with a level-one heading.
func (d *Document) InsertHeading(text string) error {
	if !Headable(text) {
		return ErrNotAHeading
	}
	rest := bytes.TrimLeft(d.body, "\r\n")
	if len(bytes.TrimSpace(rest)) == 0 {
		d.body = []byte("# " + text + d.eol)
		return nil
	}
	d.body = append([]byte("# "+text+d.eol+d.eol), rest...)
	return nil
}

// lineEnd is where the line beginning at start stops, before the break that
// ends it. A file's own break is left standing whichever one it is.
func lineEnd(body []byte, start int) int {
	end := len(body)
	if next := bytes.IndexByte(body[start:], '\n'); next >= 0 {
		end = start + next
	}
	if end > start && body[end-1] == '\r' {
		end--
	}
	return end
}
