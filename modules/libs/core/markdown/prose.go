package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// PointProseAt sends every wikilink in the prose that goes to one address to
// another, and says how many it moved.
//
// Only the target inside the brackets changes. An alias after `|` is how the
// link is read in the sentence and a fragment after `#` names a place inside
// the note; both survive.
//
// What stands inside a code fence is an example of a link, not one, and is left
// as it was written.
//
// A note whose frontmatter block is never closed holds no prose of its own —
// the whole file stands as its body, links in the block included — so this is
// ErrUnterminated and the file is left as it is.
func (d *Document) PointProseAt(from domain.Address, to string) (int, error) {
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

	for at := 0; at <= len(body); {
		end := len(body)
		if next := strings.IndexByte(body[at:], '\n'); next >= 0 {
			end = at + next
		}
		line := strings.TrimRight(body[at:end], "\r")
		if f.Cross(line) || f.IsInside() {
			at = end + 1
			continue
		}
		for _, found := range WikilinksIn(line) {
			if found.Target != from {
				continue
			}
			// The brackets stay where they were: only the target between them
			// moves, and each of them is two bytes.
			out.WriteString(body[last : at+found.At+2])
			out.WriteString(to + getAfterTarget(found.Inside))
			last = at + found.To - 2
			moved++
		}
		at = end + 1
	}
	if moved == 0 {
		return 0, nil
	}
	out.WriteString(body[last:])
	d.body = []byte(out.String())
	return moved, nil
}

// getAfterTarget is the part of a wikilink that is not the address: the
// fragment and the alias, in the order they were written.
func getAfterTarget(inside string) string {
	target := inside
	if alias := strings.IndexByte(target, '|'); alias >= 0 {
		target = target[:alias]
	}
	if fragment := strings.IndexByte(target, '#'); fragment >= 0 {
		target = target[:fragment]
	}
	return inside[len(target):]
}
