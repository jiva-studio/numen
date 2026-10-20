package run

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetHash is the fingerprint of a source's bytes: known when it is set, read
// from the vault otherwise.
func GetHash(ctx context.Context, readers port.VaultReaders, v domain.Vault, path, known string) (string, error) {
	if known != "" {
		return known, nil
	}
	reader, err := readers.Open(v)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return text.Fingerprint(raw), nil
}

// Cut makes a source's chunks through cut, and does nothing where none is set.
func Cut(ctx context.Context, cut func(context.Context, domain.Vault, string) error, v domain.Vault, path string) error {
	if cut == nil {
		return nil
	}
	return cut(ctx, v, path)
}

// Stand puts the source on the text a producer made. Cutting writes which text
// a source stands on together with its chunks; where nothing cuts, the source
// is recorded as owing its text and the scan that cuts it writes both.
func Stand(
	ctx context.Context,
	cut func(context.Context, domain.Vault, string) error,
	sources port.SourceRepository,
	v domain.Vault,
	ref domain.Fingerprint,
	hash, from string,
) error {
	if cut != nil {
		return cut(ctx, v, ref.Path)
	}
	return sources.SaveSource(ctx, v.ID, domain.Source{Fingerprint: ref, Hash: hash, Producer: from})
}

// GetArea is the store a run keeps its artifact in: the one set, else the
// default.
func GetArea(set, byDefault string) string {
	if set == "" {
		return byDefault
	}
	return set
}

// GetSize is a size a run works in: the one set, else the default.
func GetSize[T int | float64](set, byDefault T) T {
	if set <= 0 {
		return byDefault
	}
	return set
}

// Report hands a result to the run's progress callback, when it has one.
func Report[T any](on func(T), res T) {
	if on != nil {
		on(res)
	}
}
