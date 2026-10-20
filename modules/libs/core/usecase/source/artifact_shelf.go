package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

// artifactFile is the file one producer's artifact stands in, and its size.
type artifactFile struct {
	Name domain.ArtifactName
	Size int64
}

// getArtifactFile returns the file of one producer's artifact, or nil when the store holds none.
func getArtifactFile(
	ctx context.Context, store port.DerivedStore, producer, hash string,
) (*artifactFile, error) {
	for _, one := range []domain.ArtifactName{
		domain.NewArtifactName(domain.ArtifactCorrections, producer, hash),
		domain.NewArtifactName(domain.ArtifactText, producer, hash),
		domain.NewArtifactName(domain.ArtifactPartial, producer, hash),
	} {
		reader, size, err := store.OpenArtifact(ctx, one)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		_ = reader.Close()
		return &artifactFile{Name: one, Size: size}, nil
	}
	return nil, nil
}

// artifactSet is what the store holds for one file, with the store and the
// hash it was found under.
type artifactSet struct {
	Artifacts []Artifact
	Store     port.DerivedStore
	Hash      string
}

// getArtifacts is what the store holds for one file. An empty hash is a file
// the index does not hold.
func (s artifactShelf) getArtifacts(ctx context.Context, v domain.Vault, path string) (artifactSet, error) {
	hash, err := s.getHash(ctx, v, path)
	if err != nil || hash == "" {
		return artifactSet{}, err
	}
	store, err := s.derived.Open(v)
	if err != nil {
		return artifactSet{}, err
	}
	out := make([]Artifact, 0, 2)
	for _, producer := range domain.GetAddressProducers() {
		found, err := getArtifactFile(ctx, store, producer, hash)
		if err != nil {
			return artifactSet{}, err
		}
		if found == nil {
			continue
		}
		out = append(out, Artifact{
			Path:     path,
			Kind:     getArtifactKind(producer),
			Producer: producer,
			Bytes:    found.Size,
			Format:   getArtifactFormat(producer),
		})
	}
	if file, size, err := store.OpenArtifact(ctx, domain.NewCopyName(hash)); err == nil {
		_ = file.Close()
		out = append(out, Artifact{Path: path, Kind: KindCopy, Bytes: size, Format: domain.CopyType})
	} else if !errors.Is(err, fs.ErrNotExist) {
		return artifactSet{}, err
	}
	return artifactSet{Artifacts: out, Store: store, Hash: hash}, nil
}

// getArtifactWords is the artifact as it stands on disk. A transcript is the WebVTT it
// is written in, times and all.
func getArtifactWords(ctx context.Context, store port.DerivedStore, producer, hash string) (string, error) {
	found, err := getArtifactFile(ctx, store, producer, hash)
	if err != nil || found == nil {
		return "", err
	}
	raw, err := store.ReadArtifact(ctx, found.Name)
	return string(raw), err
}
