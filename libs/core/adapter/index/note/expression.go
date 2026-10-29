package note

import (
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// The two characters the index wraps around a matched run. They are control
// characters, which no user-written name contains.
const (
	opening = "\x02"
	closing = "\x03"
)

// parseMarkedName returns a marked name without its markers, and the runs the
// index marked in it, counted as a client counts text: in UTF-16 code units.
func parseMarkedName(marked string) (string, []domain.UnitRange) {
	var name strings.Builder
	name.Grow(len(marked))

	var pos []domain.UnitRange
	if n := strings.Count(marked, "\x02"); n > 0 {
		pos = make([]domain.UnitRange, 0, n)
	}

	units, from := 0, -1
	for _, r := range marked {
		switch r {
		case '\x02':
			from = units
		case '\x03':
			if from >= 0 && units > from {
				pos = append(pos, domain.UnitRange{Start: from, End: units})
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
	return name.String(), pos
}
