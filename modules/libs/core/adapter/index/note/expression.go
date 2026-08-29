package note

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The two characters the index is asked to wrap a run that matched in. They are
// control characters, which no name written by a person holds.
const (
	opening = "\x02"
	closing = "\x03"
)

// split is a marked name as it actually reads, and the runs the index marked in
// it, counted the way a client counts text: in UTF-16 code units.
func split(marked string) (string, []domain.Span) {
	var name strings.Builder
	var at []domain.Span

	units, from := 0, -1
	for _, r := range marked {
		switch r {
		case '\x02':
			from = units
		case '\x03':
			if from >= 0 && units > from {
				at = append(at, domain.Span{From: from, To: units})
			}
			from = -1
		default:
			name.WriteRune(r)
			units++
			if r > 0xffff {
				units++
			}
		}
	}
	return name.String(), at
}
