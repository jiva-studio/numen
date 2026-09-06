// Package file holds the scenarios that act on a vault's files by their path,
// whatever kind of file the vault holds each of them as.
package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	pathpkg "path"
	"path/filepath"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// MostRead is how much of a file one read carries.
const MostRead = 16000

// ReadOutcome is how a read ended.
type ReadOutcome string

const (
	// Ok is the text coming back.
	Ok ReadOutcome = "ok"
	// Missing is a path with no file behind it.
	Missing ReadOutcome = "missing"
	// LeftAlone is a file the vault's own rules pass over.
	LeftAlone ReadOutcome = "left alone"
	// AFolder is a path holding a folder.
	AFolder ReadOutcome = "a folder"
	// NotText is a run of bytes that is not valid UTF-8.
	NotText ReadOutcome = "not text"
)

// ReadResult is one run of a file as a read hands it over.
type ReadResult struct {
	Outcome ReadOutcome
	// Text is the run that was read. It is empty for every outcome but Ok.
	Text string
	// Start and Length are the run that came back: what was asked for, held
	// within the file and cut at the characters around it.
	Start  int
	Length int
	// Whole is how long the file is, in bytes, so a caller knows what stands on
	// either side of the run.
	Whole int
}

// Read hands over a run of one file's bytes, addressed by its path from the
// vault root.
//
// What the vault passes over is passed over here: the folder belonging to the
// application, and every name the vault's ignore rules match.
type Read struct {
	Readers port.VaultReaders
}

// Execute reads length bytes of the file at path, beginning at start. A length
// of zero asks for as much as one read carries.
//
// An error is the vault being out of reach, or a path that does not name a file
// inside it. What is wrong with the file itself is an outcome.
func (u Read) Execute(
	ctx context.Context, v domain.Vault, path string, start, length int,
) (ReadResult, error) {
	out := ReadResult{Start: start}
	if start < 0 {
		return out, fmt.Errorf("a run of %s begins at %d", path, start)
	}
	if length < 0 || length > MostRead {
		return out, fmt.Errorf("read between 1 and %d bytes of a file at a time", MostRead)
	}
	if length == 0 {
		length = MostRead
	}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return ReadResult{}, err
	}
	file, err := reader.Open(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = Missing
		return out, nil
	case err != nil:
		return ReadResult{}, err
	}
	defer file.Close()

	// A listing of the folder above is the vault saying what it holds there.
	held, err := reported(ctx, reader, path)
	if err != nil {
		return ReadResult{}, err
	}
	if held != Ok {
		out.Outcome = held
		return out, nil
	}

	whole, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return ReadResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	out.Whole = int(whole)

	start = min(start, out.Whole)
	length = min(length, out.Whole-start)
	if _, err := file.Seek(int64(start), io.SeekStart); err != nil {
		return ReadResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	raw := make([]byte, length)
	read, err := io.ReadFull(file, raw)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return ReadResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	raw = raw[:read]

	start, raw = wholeCharacters(start, raw, out.Whole)
	out.Start = start
	if !utf8.Valid(raw) {
		out.Outcome = NotText
		return out, nil
	}
	out.Outcome = Ok
	out.Text = string(raw)
	out.Length = len(raw)
	return out, nil
}

// reported is what the vault names at this path among the entries of the folder
// above it: Ok for a file, AFolder for a folder, LeftAlone for a path it does
// not report at all.
func reported(ctx context.Context, reader port.VaultReader, path string) (ReadOutcome, error) {
	clean := pathpkg.Clean(filepath.ToSlash(path))
	folder := pathpkg.Dir(clean)
	if folder == "." {
		folder = ""
	}
	entries, err := reader.List(ctx, folder)
	if errors.Is(err, fs.ErrNotExist) {
		return LeftAlone, nil
	}
	if err != nil {
		return LeftAlone, err
	}
	for _, entry := range entries {
		if entry.Path != clean {
			continue
		}
		if entry.IsFolder {
			return AFolder, nil
		}
		return Ok, nil
	}
	return LeftAlone, nil
}

// wholeCharacters is the run with the character it opened inside and the one it
// closed inside left off.
//
// A run reaching the end of the file closes inside nothing: an incomplete
// character there is what the file holds.
func wholeCharacters(start int, raw []byte, whole int) (int, []byte) {
	for start > 0 && len(raw) > 0 && !utf8.RuneStart(raw[0]) {
		raw = raw[1:]
		start++
	}
	if start+len(raw) >= whole {
		return start, raw
	}
	for at := len(raw) - 1; at >= 0 && at > len(raw)-utf8.UTFMax; at-- {
		if !utf8.RuneStart(raw[at]) {
			continue
		}
		if !utf8.FullRune(raw[at:]) {
			raw = raw[:at]
		}
		break
	}
	return start, raw
}
