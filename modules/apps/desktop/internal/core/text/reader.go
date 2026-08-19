package text

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// A Reader is where a source's text comes from: the file itself, or the file a
// recognition wrote.
//
// It is the one type that answers this, because a search showing a passage, an
// embedder re-slicing a window and an extractor cutting a source all ask it, and
// three answers that drift are three ways to read the wrong place.
type Reader struct {
	Vault   port.VaultReader
	Derived port.DerivedStore
}

// Of is the text a source's chunks are places in.
//
// A source naming a producer reads what that producer wrote or reads nothing.
// Falling back to the document would slice one text at another text's offsets,
// which is a wrong answer given confidently and is worse than no answer.
func (r Reader) Of(ctx context.Context, path, from, hash string) (*Document, error) {
	if from != "" {
		return r.recognised(ctx, from, hash)
	}
	ref, err := r.Vault.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	raw, err := r.Vault.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return Read(ref, raw)
}

// recognised is a source whose text a producer wrote. A recognition still
// running is the source's text while it runs.
func (r Reader) recognised(ctx context.Context, from, hash string) (*Document, error) {
	if r.Derived == nil {
		return nil, ErrUnreadable
	}
	for _, name := range []string{Artifact(from, hash), Partial(from, hash)} {
		raw, err := r.Derived.Read(ctx, name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return Recognised(raw), nil
	}
	// The store is a folder on the person's disk and they may empty it. The
	// source says nothing until a scan notices and cuts it again.
	return nil, ErrUnreadable
}

// Recognised is a recognition, as the text its chunks are places in and the
// pages that text names.
//
// A recognised document names no parts. What a layout model calls a heading is
// a shape on a page and not an entry in an outline, and turning one into the
// other would be a guess about what the book's parts are.
func Recognised(raw []byte) *Document {
	prose, marks := ocr.Read(raw)
	doc := &Document{Text: prose}
	for _, m := range marks {
		doc.paged = append(doc.paged, mark{Offset: m.Offset, Name: m.Label})
	}
	return doc
}

// Artifact is the name a producer's recognition of these bytes is kept under.
//
// It is the hash of what was read and not the path it was read from, so a
// document renamed or moved keeps its recognition, and two copies of one
// document in a vault share the one file rather than being read twice.
func Artifact(from, hash string) string {
	return from + "/" + hash + ".txt"
}

// Partial is the name a producer's recognition still running is kept under. It
// is not an artifact until it is complete, and nothing reads it back as one.
func Partial(from, hash string) string {
	return from + "/" + hash + ".partial"
}

// Boxes is the name the coordinates a model produced are kept under. They are
// kept because no machine here remakes them cheaply.
func Boxes(from, hash string) string {
	return from + "/" + hash + ".boxes"
}

// Beside is the name of what says which models produced an artifact. Nothing on
// any hot path reads it; it is there so a person can ask what read a text they
// are looking at, and so a sweep can find everything a recogniser now known to
// be bad produced.
func Beside(from, hash string) string {
	return from + "/" + hash + ".json"
}

// Names is every file one recognition of these bytes is kept under. One run
// made them and none of them means anything without the others.
func Names(from, hash string) []string {
	return []string{
		Artifact(from, hash),
		Partial(from, hash),
		Boxes(from, hash),
		Beside(from, hash),
	}
}
