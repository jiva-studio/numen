package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Wikilink is one `[[…]]` where it stands in a line.
type Wikilink struct {
	// At is the byte the brackets open at, and To the byte after they close.
	At, To int
	// Inside is what stands between them, as it was written.
	Inside string
	// Target is where it points.
	Target domain.Address
}

// WikilinksIn is every wikilink in one line, in the order they are written.
// Brackets holding nothing point nowhere, so they are the text they are
// written as.
//
// A link in prose, the stencil a card is cut by and what a rename moves all
// come through here, so the brackets answer one question and not three.
func WikilinksIn(line string) []Wikilink {
	var out []Wikilink
	pos := 0
	for pos < len(line) {
		start := strings.Index(line[pos:], "[[")
		if start < 0 {
			break
		}
		start += pos
		closeIdx := strings.Index(line[start+2:], "]]")
		if closeIdx < 0 {
			break
		}
		end := start + 2 + closeIdx
		inside := line[start+2 : end]
		if strings.ContainsAny(inside, "[]") {
			pos = start + 1
			continue
		}
		if len(inside) == 0 {
			pos = end + 2
			continue
		}
		target := domain.ParseAddress(inside)
		if target.Value != "" {
			out = append(out, Wikilink{At: start, To: end + 2, Inside: inside, Target: target})
		}
		pos = end + 2
	}
	return out
}
