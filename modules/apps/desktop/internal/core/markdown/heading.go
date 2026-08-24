package markdown

import "strings"

// SetHeading writes the text of the note's first level-one heading, and says
// whether the note has one. Only that line is replaced.
func (d *Document) SetHeading(text string) bool {
	body := Normalised(d.Body())
	for _, h := range headings([]byte(body)) {
		if h.Level != 1 {
			continue
		}
		end := len(body)
		if next := strings.IndexByte(body[h.Offset:], '\n'); next >= 0 {
			end = h.Offset + next
		}
		d.SetBody(body[:h.Offset] + "# " + text + body[end:])
		return true
	}
	return false
}

// InsertHeading opens the prose with a level-one heading.
func (d *Document) InsertHeading(text string) {
	body := strings.TrimLeft(Normalised(d.Body()), "\n")
	if strings.TrimSpace(body) == "" {
		d.SetBody("# " + text)
		return
	}
	d.SetBody("# " + text + "\n\n" + body)
}
