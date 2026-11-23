package identity

import (
	"context"
	"io"

	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// GenerateRecoveryWords makes 12 new recovery words.
type GenerateRecoveryWords struct {
	random  io.Reader
	created *generatedWords
}

// NewGenerateRecoveryWords returns the use case that makes new recovery words.
func NewGenerateRecoveryWords(d Deps) *GenerateRecoveryWords {
	return &GenerateRecoveryWords{random: d.random, created: &d.progress.created}
}

// Execute returns 12 new words and holds them in memory until CreateIdentity takes them; the next call replaces them.
func (u *GenerateRecoveryWords) Execute(_ context.Context) ([]string, error) {
	words, err := identitylib.NewRecoveryWords(u.random)
	if err != nil {
		return nil, err
	}
	listed := words.List()
	u.created.put(listed)
	return listed, nil
}
