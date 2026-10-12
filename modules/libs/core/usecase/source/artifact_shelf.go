package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// artifactShelf is where the artifacts made from a vault's files are found.
type artifactShelf struct {
	queries port.SourceQueries
	derived port.DerivedStores
}

// getHash is the hash everything made from one file stands under. A url stands
// under its address, and the index recorded that when it walked past.
func (s artifactShelf) getHash(ctx context.Context, v domain.Vault, path string) (string, error) {
	said, held, err := s.queries.GetSourceText(ctx, v.ID, path)
	if err != nil || !held {
		return "", err
	}
	return said.Hash, nil
}

// getFileName is the file one producer's artifact stands in, and its size.
func getFileName(
	ctx context.Context, store port.DerivedStore, producer, hash string,
) (name string, size int64, held bool, err error) {
	for _, one := range []string{
		text.Corrections(producer, hash),
		text.Artifact(producer, hash),
		text.Partial(producer, hash),
	} {
		reader, size, err := store.Open(ctx, one)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", 0, false, err
		}
		_ = reader.Close()
		return one, size, true, nil
	}
	return "", 0, false, nil
}

// getArtifacts is what the store holds for one file, with the store and the
// hash it was found under. An empty hash is a file the index does not hold.
func (s artifactShelf) getArtifacts(
	ctx context.Context, v domain.Vault, path string,
) ([]Artifact, port.DerivedStore, string, error) {
	hash, err := s.getHash(ctx, v, path)
	if err != nil || hash == "" {
		return nil, nil, "", err
	}
	store, err := s.derived.Open(v)
	if err != nil {
		return nil, nil, "", err
	}
	out := make([]Artifact, 0, 2)
	for _, producer := range text.Producers() {
		_, size, held, err := getFileName(ctx, store, producer, hash)
		if err != nil {
			return nil, nil, "", err
		}
		if !held {
			continue
		}
		out = append(out, Artifact{
			Path:     path,
			Kind:     getArtifactKind(producer),
			Producer: producer,
			Bytes:    size,
			Format:   getArtifactFormat(producer),
		})
	}
	if file, size, err := store.Open(ctx, text.Copy(hash)); err == nil {
		_ = file.Close()
		out = append(out, Artifact{Path: path, Kind: KindCopy, Bytes: size, Format: text.CopyType})
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, "", err
	}
	return out, store, hash, nil
}

// getArtifactWords is the artifact as it stands on disk. A transcript is the WebVTT it
// is written in, times and all.
func getArtifactWords(ctx context.Context, store port.DerivedStore, producer, hash string) (string, error) {
	name, _, held, err := getFileName(ctx, store, producer, hash)
	if err != nil || !held {
		return "", err
	}
	raw, err := store.Read(ctx, name)
	return string(raw), err
}
