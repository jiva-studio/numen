package markdown

import (
	"regexp"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A wikilink is the ordinary link people write. Anything inside the brackets is
// the target, an alias, or a fragment; the address parser sorts that out.
var wikilinkRe = regexp.MustCompile(`\[\[([^\]\[]+)\]\]`)

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
	for _, found := range wikilinkRe.FindAllStringSubmatchIndex(line, -1) {
		inside := line[found[2]:found[3]]
		target := domain.ParseAddress(inside)
		if target.Value == "" {
			continue
		}
		out = append(out, Wikilink{At: found[0], To: found[1], Inside: inside, Target: target})
	}
	return out
}
