package text

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/correction"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// The producers: what made a text, which is not the same question as what the
// text is. A kind with one producer is named by it, and a kind with two names
// both in the files.
const (
	// ASR is a model here, listening to a recording.
	ASR = "asr"
	// Reading is a model here, reading the pages of a scan.
	Reading = "ocr"
)

// The producers that bring back what is at an address a link note points at.
const (
	// Captions is a site, publishing words with a video.
	Captions = "captions"
	// Article is the reader that finds the prose a page is written around.
	Article = "article"
)

// timed says whether a producer writes words with the times they were said at.
// Those are WebVTT and open in a player; everything else is prose.
func timed(producer string) bool { return producer == ASR || producer == Captions }

// Transcript is where text with the times it was said at is kept, and Copies
// where the bytes of a video are.
//
// A folder is a kind: what the files in it are. A producer is what made them,
// which is another question — a transcript is a transcript whether a model here
// heard it or a site published it with the video, so both stand here under
// names of their own.
const (
	Transcript = "transcript"
	Copies     = "copy"
)

// kind is the folder a producer's files stand in, which is what they are. A
// kind one producer writes is that producer's own name: there is nothing to
// tell its files apart from.
func kind(producer string) string {
	if timed(producer) {
		return Transcript
	}
	return producer
}

// under is the name one of a producer's files stands under inside its kind: the
// hash, then the producer where the kind has more than one, then what the file
// is.
func under(producer, hash, what string) string {
	if timed(producer) {
		return Transcript + "/" + hash + "." + producer + what
	}
	return producer + "/" + hash + what
}

// The container a copy of a video is fetched and kept in, and what a player is
// told it is. One is asked for and one is kept, so what a player is handed is
// what every player this window is drawn in opens.
const (
	CopyExtension = ".mp4"
	CopyType      = "video/mp4"
)

// Copy is the name a copy of what is at an address is kept under. It is not
// text and nothing reads it as any: it is the bytes a person plays, kept where
// they can be fetched again from the address the note carries.
func Copy(hash string) string { return Copies + "/" + hash + CopyExtension }

// Separator is what stands between a link note's own prose and what was fetched
// for it. The two are one text, and a chunk is cut across neither into the
// other.
const Separator = "\n\n"

// A Reader is where a source's text comes from: the file itself, or the file a
// recognition wrote.
//
// It is the one type that answers this, because a search showing a passage, an
// embedder re-slicing a chunk and an extractor cutting a source all ask it, and
// three answers that drift are three ways to read the wrong place.
type Reader struct {
	Vault   port.VaultReader
	Derived port.DerivedStore
	// Documents reads a format that needs a library. A vault holding none is
	// read without one.
	Documents port.TextExtractor
}

// Of is the text a source's chunks are places in.
//
// A source naming a producer reads what that producer wrote or reads nothing.
// Falling back to the document would slice one text at another text's offsets,
// which is a wrong answer given confidently and is worse than no answer.
func (r Reader) Of(ctx context.Context, path, producer, hash string) (*Document, error) {
	if producer != "" {
		// A note naming a producer is a link: what a person wrote and what was
		// fetched for the address they wrote it about are one text, and an
		// offset in it falls in whichever of the two it lands in.
		if strings.HasSuffix(path, domain.NoteExtension) {
			return r.pointed(ctx, path, producer, hash)
		}
		return r.recognised(ctx, producer, hash)
	}
	ref, err := r.Vault.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	raw, err := r.Vault.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return Read(ctx, r.Documents, ref, raw)
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

// pointed is a link note: the prose its person wrote, and what was fetched from
// the address it points at, as one text.
//
// The prose comes first because it is what the person opened the note to write.
// A note whose fetch brought back nothing is its prose alone, and one nothing
// has fetched for yet is the same.
func (r Reader) pointed(ctx context.Context, path, producer, hash string) (*Document, error) {
	raw, err := r.Vault.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	doc := &Document{Text: markdown.Body(raw)}
	fetched, err := r.recognised(ctx, producer, hash)
	if errors.Is(err, ErrUnreadable) {
		return doc, nil
	}
	if err != nil {
		return nil, err
	}
	return Joined(doc.Text, fetched), nil
}

// Joined is a link note's prose and what was fetched for it, as one text.
//
// What the fetch named — the moment a stretch of speech was said, the parts of
// an article — is moved out by as much as the prose and what stands between
// them, so a place in the fetched half is the place it was.
func Joined(prose string, fetched *Document) *Document {
	at := len(prose) + len(Separator)
	doc := &Document{Text: prose + Separator + fetched.Text}
	for _, part := range fetched.Parts {
		part.Offset += at
		doc.Parts = append(doc.Parts, part)
	}
	for _, one := range fetched.named {
		doc.named = append(doc.named, namedPlace{Offset: one.Offset + at, Name: one.Name})
	}
	for _, one := range fetched.paged {
		doc.paged = append(doc.paged, namedPlace{Offset: one.Offset + at, Name: one.Name})
	}
	return doc
}

// Producers are the ones that bring back what is at an address, in the order
// one of them is read: what a model here heard stands over what a site
// published, because listening is asked for and publishing is not.
func Producers() []string { return []string{ASR, Captions, Article} }

// Fetched is what was brought back for an address, as the text a link note is
// cut with, and which producer brought it.
//
// Nothing fetched is no text and no producer, which is a link note nothing has
// been fetched for and is its ordinary state until something is.
func Fetched(
	ctx context.Context, store port.DerivedStore, hash string,
) (words, producer string, err error) {
	if store == nil {
		return "", "", nil
	}
	for _, from := range Producers() {
		for _, name := range []string{Artifact(from, hash), Partial(from, hash)} {
			raw, err := store.Read(ctx, name)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return "", "", err
			}
			doc, err := Composed(ctx, store, from, hash, raw)
			if err != nil {
				return "", "", err
			}
			return doc.Text, from, nil
		}
	}
	return "", "", nil
}

// Composed is a reading and everything kept beside it, as the text a source's
// chunks are places in.
//
// The corrections are read before the coordinates, and a reading nothing
// proofread is composed from its own bytes alone. A transcript is composed from
// what it was put right to, and from its own bytes where nothing put it right.
func Composed(
	ctx context.Context,
	store port.DerivedStore,
	producer, hash string,
	raw []byte,
) (*Document, error) {
	if timed(producer) {
		put, err := beside(ctx, store, Corrections(producer, hash))
		if err != nil {
			return nil, err
		}
		// A file beside the artifact holding no words is nothing put right, and
		// the recording says what was heard in it.
		if doc := Transcribed(put); doc.Text != "" {
			return doc, nil
		}
		return Transcribed(raw), nil
	}
	parts, err := beside(ctx, store, Parts(producer, hash))
	if err != nil {
		return nil, err
	}
	corrections, err := beside(ctx, store, Corrections(producer, hash))
	if err != nil {
		return nil, err
	}
	var boxes []byte
	if len(corrections) > 0 {
		if boxes, err = beside(ctx, store, Boxes(producer, hash)); err != nil {
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
	if put := correction.Unpack(corrections); len(put) > 0 {
		prose, marks, named = correction.Prose(prose, marks, highlight.Unpack(boxes), named, put)
	}
	doc := &Document{Text: prose}
	for _, p := range divided(prose, named) {
		doc.Parts = append(doc.Parts, p)
		doc.named = append(doc.named, namedPlace{Offset: p.Offset, Name: p.Title})
	}
	for i, m := range marks {
		doc.paged = append(doc.paged, namedPlace{Offset: m.Offset, Name: page(i)})
	}
	return doc
}

// Transcribed is what a model heard, as the text its chunks are places in and
// the moments of the recording those places stand at.
//
// A transcript names no parts: the cues are where the speech was, and a chunk
// is located by when what it holds was said.
func Transcribed(raw []byte) *Document {
	prose, cues := transcript.Parse(raw)
	doc := &Document{Text: prose}
	for _, cue := range cues {
		doc.paged = append(doc.paged, namedPlace{Offset: cue.Offset, Name: transcript.Clock(cue.From)})
	}
	return doc
}

// divided is the parts a sidecar names, as parts of the prose. A part is named
// by its heading run as the scan was read, mangled or not.
//
// The parts of one artifact begin in the order the prose is read and end within
// it. A sidecar that says otherwise was written for other bytes, and none of it
// is used.
func divided(prose string, parts []ocr.Part) []chunking.PartStart {
	out := make([]chunking.PartStart, 0, len(parts))
	at := 0
	for _, p := range parts {
		if p.Start < at || p.Length <= 0 || p.Start+p.Length > len(prose) {
			return nil
		}
		at = p.Start
		out = append(out, chunking.PartStart{
			Title:  prose[p.Start : p.Start+p.Length],
			Offset: p.Start,
		})
	}
	return out
}

// Fingerprint addresses the content of a file. Everything one run wrote about
// those bytes is kept under it, and whoever asks about them works it out the
// same way.
func Fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Artifact is the name a producer's recognition of these bytes is kept under.
//
// It is the hash of what was read and not the path it was read from, so a
// document renamed or moved keeps its recognition, and two copies of one
// document in a vault share the one file.
//
// The extension is the producer's: a transcript is WebVTT and opens in a player
// under the name a player knows it by.
func Artifact(from, hash string) string {
	if timed(from) {
		return under(from, hash, ".vtt")
	}
	return under(from, hash, ".txt")
}

// Partial is the name a producer's recognition still running is kept under. It
// is not an artifact until it is complete, and nothing reads it back as one.
func Partial(from, hash string) string {
	if timed(from) {
		return under(from, hash, ".partial.vtt")
	}
	return under(from, hash, ".partial")
}

// Corrections is the name what put a producer's text right is kept under. A
// text nothing proofread and nobody edited has no such file.
//
// The artifact stays what was read or heard, so deleting this file gives that
// back. A transcript's corrections are the words as they now stand, WebVTT
// under the extension that format is opened by; a reading's are one record to
// a line put right, keyed by the box the line was read from.
func Corrections(from, hash string) string {
	if timed(from) {
		return under(from, hash, ".corrected.vtt")
	}
	return under(from, hash, ".corrected")
}

// Parts is the name the parts of a reading are kept under. A reading whose
// layout model named none has no such file.
func Parts(from, hash string) string { return under(from, hash, ".parts") }

// Boxes is the name the coordinates a model produced are kept under. They are
// kept because no machine here remakes them cheaply.
func Boxes(from, hash string) string { return under(from, hash, ".boxes") }

// Proofread is the name of what says who put a reading right and how far they
// got. A run stopped part way is taken up again at the page it names.
func Proofread(from, hash string) string { return under(from, hash, ".proofread") }

// Answer is the name of what a recording gave where it gave no words: silence,
// or bytes nothing here can open. It is not a transcript and nothing reads it as
// one; it is there so that a recording nothing can be heard in is not listened
// to again every time the vault is scanned.
func Answer(from, hash string) string { return under(from, hash, ".answer") }

// The two answers a recording gives that carry no words: it holds no speech, or
// nothing here opens it. What is kept under Answer opens with one of them.
const (
	Silent   = "silent"
	Unopened = "unopened"
)

// Answered is which of the two a recording gave and what the run said about it,
// read from what is kept under Answer. Bytes opening with neither word are
// nothing this wrote.
func Answered(raw []byte) (gave, said string) {
	line := strings.TrimSpace(string(raw))
	for _, one := range []string{Silent, Unopened} {
		if line == one {
			return one, ""
		}
		if rest, cut := strings.CutPrefix(line, one+": "); cut {
			return one, rest
		}
	}
	return "", ""
}

// Beside is the name of what says which models produced an artifact. Nothing on
// any hot path reads it; it is there so a person can ask what read a text they
// are looking at, and so a sweep can find everything a recogniser now known to
// be bad produced.
func Beside(from, hash string) string { return under(from, hash, ".json") }

// Names is every file one recognition of these bytes is kept under. One run
// made them and none of them means anything without the others.
//
// Each producer's own files are named: a sweep works through this list, and a
// transcription writes no coordinates or parts.
func Names(from, hash string) []string {
	if timed(from) {
		return []string{
			Artifact(from, hash),
			Partial(from, hash),
			Corrections(from, hash),
			Proofread(from, hash),
			Answer(from, hash),
			Beside(from, hash),
		}
	}
	return []string{
		Artifact(from, hash),
		Partial(from, hash),
		Boxes(from, hash),
		Parts(from, hash),
		Corrections(from, hash),
		Proofread(from, hash),
		Beside(from, hash),
	}
}

// AddressNames is every file kept for one address: what each producer that
// reaches one writes, and the copy of the video where a person asked for one.
//
// A sweep works through this list, and it is the one place the copy is named
// beside the rest — it is the bytes of a video and no producer's text, so
// Names, which is a producer's own files, does not carry it.
func AddressNames(hash string) []string {
	return append([]string{Copy(hash)}, AddressTexts(hash)...)
}

// AddressTexts is every file a producer wrote for one address, without the
// copy.
func AddressTexts(hash string) []string {
	var out []string
	for _, producer := range Producers() {
		out = append(out, Names(producer, hash)...)
	}
	return out
}
