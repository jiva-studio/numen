package identity

import (
	"context"
	"io"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// GenerateRecoveryWords makes 12 new recovery words.
type GenerateRecoveryWords struct{ random io.Reader }

// NewGenerateRecoveryWords returns the use case that makes new recovery words.
func NewGenerateRecoveryWords(d Deps) *GenerateRecoveryWords {
	return &GenerateRecoveryWords{random: d.random}
}

// Execute returns 12 new words and keeps nothing.
func (u *GenerateRecoveryWords) Execute(_ context.Context) ([]string, error) {
	words, err := identitylib.NewRecoveryWords(u.random)
	if err != nil {
		return nil, err
	}
	return words.List(), nil
}
