package cards

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// DeckVersion is a deck as the caller read it. A write over a deck that has
// changed since is refused.
type DeckVersion struct {
	Path        string
	Fingerprint domain.Fingerprint
}
