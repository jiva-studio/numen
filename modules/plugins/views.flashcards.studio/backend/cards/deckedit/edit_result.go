package deckedit

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

type EditResult struct {
	Fingerprint domain.Fingerprint
	Cards       int
}
