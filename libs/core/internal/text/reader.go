package text

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/libs/core/internal/correction"
	"github.com/jiva-studio/numen/libs/core/internal/highlight"
	"github.com/jiva-studio/numen/libs/core/internal/ocrartifact"
	"github.com/jiva-studio/numen/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ReadDownloaded is what was brought back for an address, as the text a link
// note is cut with, and which producer brought it.
//
// Nothing downloaded is no text and no producer, which is a link note nothing
// has been downloaded for and is its ordinary state until something is.
func ReadDownloaded(
	ctx context.Context, store vaultport.DerivedStore, hash string,
) (words, producer string, err error) {
	if store == nil {
		return "", "", nil
	}
	for _, from := range domain.GetAddressProducers() {
		for _, name := range []domain.ArtifactName{
			domain.NewArtifactName(domain.ArtifactText, from, hash),
			domain.NewArtifactName(domain.ArtifactPartial, from, hash),
		} {
			raw, err := store.ReadArtifact(ctx, name)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return "", "", err
			}
			doc, err := ReadComposed(ctx, store, from, hash, raw)
			if err != nil {
				return "", "", err
			}
			return doc.Text, from, nil
		}
	}
	return "", "", nil
}

// ReadComposed is a recognition and everything kept beside it, as the text a
// source's chunks are places in.
//
// The corrections are read before the coordinates, and a recognition nothing
// proofread is composed from its own bytes alone. A transcript is composed from
// what it was put right to, and from its own bytes where nothing put it right.
func ReadComposed(
	ctx context.Context,
	store vaultport.DerivedStore,
	producer, hash string,
	raw []byte,
) (*Document, error) {
	if domain.IsTimedProducer(producer) {
		placed, err := readOptional(ctx, store, domain.NewArtifactName(domain.ArtifactCorrections, producer, hash))
		if err != nil {
			return nil, err
		}
		// A file beside the artifact holding no words is nothing put right, and
		// the recording says what was heard in it.
		if doc := ReadTranscript(placed); doc.Text != "" {
			return doc, nil
		}
		return ReadTranscript(raw), nil
	}
	parts, err := readOptional(ctx, store, domain.NewArtifactName(domain.ArtifactParts, producer, hash))
	if err != nil {
		return nil, err
	}
	corrections, err := readOptional(ctx, store, domain.NewArtifactName(domain.ArtifactCorrections, producer, hash))
	if err != nil {
		return nil, err
	}
	var boxes []byte
	if len(corrections) > 0 {
		if boxes, err = readOptional(ctx, store, domain.NewArtifactName(domain.ArtifactBoxes, producer, hash)); err != nil {
			return nil, err
		}
	}
	return ReadRecognition(raw, parts, boxes, corrections), nil
}

// readOptional is what is kept under a name, and nothing where the store holds
// nothing.
func readOptional(ctx context.Context, store vaultport.DerivedStore, name domain.ArtifactName) ([]byte, error) {
	raw, err := store.ReadArtifact(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// ReadRecognition is a recognition, as the text its chunks are places in, the
// parts that text is divided into, and the pages it names.
//
// The parts are what a layout model called a heading, written beside the
// artifact when it was read. A recognition that names none is a document with
// no parts, and is located by its pages alone.
//
// A recognition that was proofread is composed with its corrections in it, and the
// pages and the parts stand where they now are.
func ReadRecognition(raw, parts, boxes, corrections []byte) *Document {
	prose, marks := ocrartifact.Read(raw)
	named := ocrartifact.Unpack(parts)
	if correctionsList := correction.Unpack(corrections); len(correctionsList) > 0 {
		prose, marks, named = domain.GetCorrectedProse(prose, marks, highlight.Unpack(boxes), named, correctionsList)
	}
	doc := &Document{Text: prose}
	for _, p := range getPartStarts(prose, named) {
		doc.Parts = append(doc.Parts, p)
		doc.named = append(doc.named, namedPlace{Offset: p.Offset, Name: p.Title})
	}
	for i, m := range marks {
		doc.paged = append(doc.paged, namedPlace{Offset: m.Offset, Name: page(i)})
	}
	return doc
}

// ReadTranscript is what a model heard, as the text its chunks are places in
// and the moments of the recording those places stand at.
//
// A transcript names no parts: the cues are where the speech was, and a chunk
// is located by when what it holds was said.
func ReadTranscript(raw []byte) *Document {
	prose, cues := transcript.Parse(raw)
	doc := &Document{Text: prose}
	for _, cue := range cues {
		doc.paged = append(doc.paged, namedPlace{Offset: cue.Offset, Name: transcript.Clock(cue.From)})
	}
	return doc
}

// getPartStarts is the parts a sidecar names, as parts of the prose. A part is
// named by its heading run as the scan was read, mangled or not.
//
// The parts of one artifact begin in the order the prose is read and end within
// it. A sidecar that says otherwise was written for other bytes, and none of it
// is used.
func getPartStarts(prose string, parts []domain.OCRPart) []domain.PartStart {
	out := make([]domain.PartStart, 0, len(parts))
	pos := 0
	for _, p := range parts {
		if p.Start < pos || p.Length <= 0 || p.Start+p.Length > len(prose) {
			return nil
		}
		pos = p.Start
		out = append(out, domain.PartStart{
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

// GetFingerprint is Fingerprint of everything the reader holds, read a block at
// a time.
func GetFingerprint(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// The two answers a recording gives that carry no words: it holds no speech, or
// nothing here opens it. The answer a run keeps opens with one of them.
const (
	Silent   = "silent"
	Unopened = "unopened"
)

// ReadAnswer is which of the two a recording gave and what the run said about
// it, read from the answer a run kept. Bytes opening with neither word are
// nothing this wrote.
func ReadAnswer(raw []byte) (kind, detail string) {
	line := strings.TrimSpace(string(raw))
	for _, one := range []string{Silent, Unopened} {
		if line == one {
			return one, ""
		}
		if rest, hasPrefix := strings.CutPrefix(line, one+": "); hasPrefix {
			return one, rest
		}
	}
	return "", ""
}
