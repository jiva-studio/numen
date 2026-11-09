package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// RedirectProse redirects every wikilink in the prose that targets one address
// to another, and returns how many it changed.
// Only the target inside the brackets changes; an alias after `|` and a fragment
// after `#` are preserved, and so is everything else.
// A note whose frontmatter block is never closed returns ErrUnterminated and is unchanged.
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

	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at pos == len(body) the remaining line is empty and holds no wikilinks"
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
			// The brackets keep their position: only the target between them
			// moves, and each of them is two bytes.
			out.WriteString(body[last : pos+found.At+2])
			out.WriteString(to + getWikilinkSuffix(found.Inside))
			last = pos + found.To - 2
			moved++
		}
		pos = end + 1
	}
	// gomutants:disable-next-line BRANCH_IF,INTEGER_DECREMENT reason="when moved is 0 last is 0 and writing body[0:] leaves d.body unchanged"
	if moved == 0 {
		return 0, nil
	}
	out.WriteString(body[last:])
	d.body = []byte(out.String())
	return moved, nil
}

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
