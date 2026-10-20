package cards

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// EditResult is the fingerprint a deck stands at after an edit and the number of cards it holds.
type EditResult struct {
	Fingerprint domain.Fingerprint
	Cards       int
}
