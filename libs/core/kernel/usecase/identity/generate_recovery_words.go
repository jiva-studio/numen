package identity

import (
	"context"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// GenerateRecoveryWords makes 12 new recovery words.
type GenerateRecoveryWords struct{ deps Deps }

// NewGenerateRecoveryWords returns the use case that makes new recovery words.
func NewGenerateRecoveryWords(d Deps) *GenerateRecoveryWords { return &GenerateRecoveryWords{deps: d} }

// Execute returns 12 new words and keeps nothing.
func (u *GenerateRecoveryWords) Execute(_ context.Context) ([]string, error) {
	words, err := identitylib.NewRecoveryWords(u.deps.random)
	if err != nil {
		return nil, err
	}
	return words.List(), nil
}
