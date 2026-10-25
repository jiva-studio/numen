package deckedit

import "github.com/jiva-studio/numen/libs/notes/domain"

type EditResult struct {
	Fingerprint domain.Fingerprint
	Cards       int
}
