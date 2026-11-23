package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// SeededCreate creates an identity of any words: it first hands them to the store of generated words.
type SeededCreate struct {
	deps  Deps
	inner *CreateIdentity
}

func NewSeededCreate(d Deps) *SeededCreate {
	return &SeededCreate{deps: d, inner: NewCreateIdentity(d)}
}

func (s *SeededCreate) Execute(ctx context.Context, words []string) (identitydomain.Identity, error) {
	s.deps.progress.created.put(words)
	return s.inner.Execute(ctx, words)
}
