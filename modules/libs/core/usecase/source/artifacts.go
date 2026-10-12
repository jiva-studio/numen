package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The kinds an artifact is asked for by, which are the kinds the store files
// them under.
//
// A kind says what a thing is, and a producer says what made it. Words with the
// times they were said at are a transcript whether a model heard them or a site
// published them. An ocr carries where on which page each word stands, and the
// prose a page is written around carries no such thing, so they are two kinds.
const (
	KindRecognition = text.OCR
	KindTranscript  = text.Transcript
	KindArticle     = text.Article
	KindCopy        = text.Copies
)

// Artifact is one thing made from a file of the vault. It is addressed by the
// file it was made from and the kind it is.
type Artifact struct {
	Path     string
	Kind     string
	Producer string
	Bytes    int64
	// Format is the media type the artifact is read back as.
	Format string
}

// IsReadable says whether the artifact is words to read. A copy is bytes to play.
func (a Artifact) IsReadable() bool { return a.Kind != KindCopy }

// getArtifactFormat is the media type a producer's artifact is written as.
func getArtifactFormat(producer string) string {
	switch producer {
	case text.ASR, text.Captions:
		return "text/vtt"
	case text.OCR:
		return "application/json"
	default:
		return "text/plain"
	}
}

// getArtifactKind is the kind a producer's text is.
func getArtifactKind(producer string) string {
	switch producer {
	case text.ASR, text.Captions:
		return KindTranscript
	case text.Article:
		return KindArticle
	default:
		return KindRecognition
	}
}

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

// getArtifactText is the artifact as it stands on disk. A transcript is the WebVTT it
// is written in, times and all.
func getArtifactText(ctx context.Context, store port.DerivedStore, producer, hash string) (string, error) {
	name, _, held, err := getFileName(ctx, store, producer, hash)
	if err != nil || !held {
		return "", err
	}
	raw, err := store.Read(ctx, name)
	return string(raw), err
}

// ListArtifacts lists what was made from one file of the vault: the text read
// out of a scan, a recording transcribed, the words a site published with a
// video, the prose of a page and a copy of a video.
type ListArtifacts struct {
	shelf artifactShelf
}

// NewListArtifacts lists artifacts through the index that knows each file's
// hash and the store the artifacts are kept in.
func NewListArtifacts(queries port.SourceQueries, derived port.DerivedStores) *ListArtifacts {
	return &ListArtifacts{shelf: artifactShelf{queries: queries, derived: derived}}
}

// Execute returns the artifacts of one file, none for a file the index does not hold.
func (u *ListArtifacts) Execute(ctx context.Context, v domain.Vault, path string) ([]Artifact, error) {
	held, _, _, err := u.shelf.getArtifacts(ctx, v, path)
	return held, err
}

// MostArtifactRead is how much of an artifact one read carries.
const MostArtifactRead = MostDocumentText

// ArtifactRange is a range of an artifact's text. Start is what was asked for
// held within the text, and Length is how long the range is.
type ArtifactRange struct {
	Artifact
	Words  string
	Start  int
	Length int
}

// ReadArtifact reads a range of one artifact as it stands on disk.
type ReadArtifact struct {
	shelf artifactShelf
}

// NewReadArtifact reads artifacts through the index that knows each file's
// hash and the store the artifacts are kept in.
func NewReadArtifact(queries port.SourceQueries, derived port.DerivedStores) *ReadArtifact {
	return &ReadArtifact{shelf: artifactShelf{queries: queries, derived: derived}}
}

// Execute reads the range of the artifact of the given kind. A length of zero
// or less takes as much as one answer carries, and the range is cut on whole
// characters.
func (u *ReadArtifact) Execute(
	ctx context.Context, v domain.Vault, path, kind string, start, length int,
) (ArtifactRange, error) {
	held, store, hash, err := u.shelf.getArtifacts(ctx, v, path)
	if err != nil {
		return ArtifactRange{}, err
	}
	for _, one := range held {
		if one.Kind != kind || !one.IsReadable() {
			continue
		}
		words, err := getArtifactText(ctx, store, one.Producer, hash)
		if err != nil {
			return ArtifactRange{}, err
		}
		at, run := getArtifactRange(words, start, length)
		return ArtifactRange{Artifact: one, Words: run, Start: at, Length: len(run)}, nil
	}
	return ArtifactRange{}, fmt.Errorf("nothing of the kind %q was made from %s", kind, path)
}

// getArtifactRange is the range of a text a call asked for, held within it and
// cut on whole characters.
func getArtifactRange(words string, start, length int) (int, string) {
	if length <= 0 || length > MostArtifactRead {
		length = MostArtifactRead
	}
	start = max(0, min(start, len(words)))
	length = min(length, len(words)-start)
	at, raw := text.WholeCharacters(start, []byte(words[start:start+length]), len(words))
	return at, string(raw)
}
