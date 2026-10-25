package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// RedirectProse sends every wikilink in the prose that goes to one address to
// another, and says how many it moved.
//
// Only the target inside the brackets changes. An alias after `|` is how the
// link is read in the sentence and a fragment after `#` names a place inside
// the note; both survive.
//
// as it was written.
//
// A note whose frontmatter block is never closed holds no prose of its own —
// ErrUnterminated and the file is left as it is.
func (d *Document) RedirectProse(from domain.Address, to string) (int, error) {
	if d.isUnterminated {
		return 0, ErrUnterminated
	}
	if !domain.IsNameable(to) {
		return 0, nil
	}
	var out strings.Builder
	var f Fence
	body := string(d.body)
	last, moved := 0, 0

	for pos := 0; pos <= len(body); {
		end := len(body)
		if next := strings.IndexByte(body[pos:], '\n'); next >= 0 {
			end = pos + next
		}
		line := strings.TrimRight(body[pos:end], "\r")
		if f.IsCrossedBy(line) || f.IsInside() {
			pos = end + 1
			continue
		}
		for _, found := range WikilinksIn(line) {
			if found.Target != from {
				continue
			}
			// The brackets stay where they were: only the target between them
			// moves, and each of them is two bytes.
			out.WriteString(body[last : pos+found.At+2])
			out.WriteString(to + getWikilinkSuffix(found.Inside))
			last = pos + found.To - 2
			moved++
		}
		pos = end + 1
	}
	if moved == 0 {
		return 0, nil
	}
	out.WriteString(body[last:])
	d.body = []byte(out.String())
	return moved, nil
}

// getWikilinkSuffix is the part of a wikilink that is not the address: the
// fragment and the alias, in the order they were written.
func getWikilinkSuffix(inside string) string {
	target := inside
	if alias := strings.IndexByte(target, '|'); alias >= 0 {
		target = target[:alias]
	}
	if fragment := strings.IndexByte(target, '#'); fragment >= 0 {
		target = target[:fragment]
	}
	return inside[len(target):]
}
