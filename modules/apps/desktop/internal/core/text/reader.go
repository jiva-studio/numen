package text

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/fixes"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/placed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
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
		return Composed(ctx, r.Derived, from, hash, raw)
	}
	// The store is a folder on the person's disk and they may empty it. The
	// source says nothing until a scan notices and cuts it again.
	return nil, ErrUnreadable
}

// Composed is a reading and everything kept beside it, as the text a source's
// chunks are places in.
//
// The corrections are read before the coordinates, and a reading nothing
// proofread is composed from its own bytes alone.
func Composed(
	ctx context.Context,
	store port.DerivedStore,
	from, hash string,
	raw []byte,
) (*Document, error) {
	parts, err := beside(ctx, store, Parts(from, hash))
	if err != nil {
		return nil, err
	}
	corrections, err := beside(ctx, store, Fixes(from, hash))
	if err != nil {
		return nil, err
	}
	var boxes []byte
	if len(corrections) > 0 {
		if boxes, err = beside(ctx, store, Boxes(from, hash)); err != nil {
			return nil, err
		}
	}
	return Recognised(raw, parts, boxes, corrections), nil
}

// beside is what is kept under a name, and nothing where the store holds
// nothing.
func beside(ctx context.Context, store port.DerivedStore, name string) ([]byte, error) {
	raw, err := store.Read(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Recognised is a recognition, as the text its chunks are places in, the parts
// that text is divided into, and the pages it names.
//
// The parts are what a layout model called a heading, written beside the
// artifact when it was read. A recognition that names none is a document with
// no parts, and is located by its pages alone.
//
// A reading that was proofread is composed with its corrections in it, and the
// pages and the parts stand where they now are.
func Recognised(raw, parts, boxes, corrections []byte) *Document {
	prose, marks := ocr.Read(raw)
	named := ocr.Unpack(parts)
	if put := fixes.Unpack(corrections); len(put) > 0 {
		prose, marks, named = fixes.Prose(prose, marks, placed.Unpack(boxes), named, put)
	}
	doc := &Document{Text: prose}
	for _, p := range divided(prose, named) {
		doc.Places = append(doc.Places, p)
		doc.named = append(doc.named, mark{Offset: p.Offset, Name: p.Title})
	}
	for i, m := range marks {
		doc.paged = append(doc.paged, mark{Offset: m.Offset, Name: sheet(i)})
	}
	return doc
}

// divided is the parts a sidecar names, as places in the prose. A part is named
// by its heading run as the scan was read, mangled or not.
//
// The parts of one artifact begin in the order the prose is read and end within
// it. A sidecar that says otherwise was written for other bytes, and none of it
// is used.
func divided(prose string, parts []ocr.Part) []window.Place {
	places := make([]window.Place, 0, len(parts))
	at := 0
	for _, p := range parts {
		if p.Start < at || p.Length <= 0 || p.Start+p.Length > len(prose) {
			return nil
		}
		at = p.Start
		places = append(places, window.Place{
			Title:  prose[p.Start : p.Start+p.Length],
			Offset: p.Start,
		})
	}
	return places
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

// Parts is the name the parts of a reading are kept under. A reading whose
// layout model named none has no such file.
func Parts(from, hash string) string {
	return from + "/" + hash + ".parts"
}

// Boxes is the name the coordinates a model produced are kept under. They are
// kept because no machine here remakes them cheaply.
func Boxes(from, hash string) string {
	return from + "/" + hash + ".boxes"
}

// Fixes is the name a reading's corrections are kept under. A reading nothing
// proofread has no such file.
func Fixes(from, hash string) string {
	return from + "/" + hash + ".fixes"
}

// Proofread is the name of what says who put a reading right and how far they
// got. A run stopped part way is taken up again at the page it names.
func Proofread(from, hash string) string {
	return from + "/" + hash + ".proofread"
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
		Parts(from, hash),
		Fixes(from, hash),
		Proofread(from, hash),
		Beside(from, hash),
	}
}
