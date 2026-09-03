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
func (d *Document) PointProseAt(from domain.Address, to string) int {
	if !domain.Nameable(to) {
		return 0
	}
	var out strings.Builder
	var f fence
	body := string(d.body)
	last, moved := 0, 0

	for at := 0; at <= len(body); {
		end := len(body)
		if next := strings.IndexByte(body[at:], '\n'); next >= 0 {
			end = at + next
		}
		line := strings.TrimRight(body[at:end], "\r")
		if f.crosses(line) || f.inside() {
			at = end + 1
			continue
		}
		for _, found := range wikilinkRe.FindAllStringSubmatchIndex(line, -1) {
			inside := line[found[2]:found[3]]
			if domain.ParseAddress(inside) != from {
				continue
			}
			out.WriteString(body[last : at+found[2]])
			out.WriteString(to + keptAfterTarget(inside))
			last = at + found[3]
			moved++
		}
		at = end + 1
	}
	if moved == 0 {
		return 0
	}
	out.WriteString(body[last:])
	d.body = []byte(out.String())
	return moved
}

// keptAfterTarget is the part of a wikilink that is not the address: the
// fragment and the alias, in the order they were written.
func keptAfterTarget(inside string) string {
	target := inside
	if alias := strings.IndexByte(target, '|'); alias >= 0 {
		target = target[:alias]
	}
	if fragment := strings.IndexByte(target, '#'); fragment >= 0 {
		target = target[:fragment]
	}
	return inside[len(target):]
}
