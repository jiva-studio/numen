package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// PointProseAt sends every wikilink in the prose that goes to one address to
// another, and says how many it moved.
//
// Only the target inside the brackets changes. An alias after `|` is how the
// link is read in the sentence and a fragment after `#` names a place inside
// the note; both survive.
func (d *Document) PointProseAt(from domain.Address, to string) int {
	if !domain.Nameable(to) {
		return 0
	}
	var out strings.Builder
	body := string(d.body)
	last, moved := 0, 0

	for _, at := range wikilinkRe.FindAllStringSubmatchIndex(body, -1) {
		inside := body[at[2]:at[3]]
		if domain.ParseAddress(inside) != from {
			continue
		}
		out.WriteString(body[last:at[2]])
		out.WriteString(to + keptAfterTarget(inside))
		last = at[3]
		moved++
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
