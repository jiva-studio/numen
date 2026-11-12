// Package file holds the scenarios that act on a vault's files by path,
// regardless of the type the vault classifies each file as.
package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	pathpkg "path"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// MostRead is how much of a file one read carries.
const MostRead = 16000

// ReadOutcome is how a read ended.
type ReadOutcome string

const (
	// Ok is the returned text.
	Ok ReadOutcome = "ok"
	// Missing is a path with no file behind it.
	Missing ReadOutcome = "missing"
	// LeftAlone is a file the vault's own rules pass over.
	LeftAlone ReadOutcome = "left alone"
	// AFolder is a path holding a folder.
	AFolder ReadOutcome = "a folder"
	// NotText is a range of bytes that is not valid UTF-8.
	NotText ReadOutcome = "not text"
)

// ReadResult is one range of a file returned by a read.
type ReadResult struct {
	Outcome ReadOutcome
	Text    string
	Start   int
	Length  int
	Whole   int
}

// Read returns a range of one file's bytes, addressed by its path from the
// vault root.
//
// Paths the vault excludes are excluded here: the application's own folder and
// every name matched by the vault's ignore rules.
type Read struct {
	Readers vaultport.VaultReaders
}

// Execute reads length bytes of the file at path, beginning at start. A length
// of zero requests the maximum a single read returns.
//
// An error means the vault is unreachable or the path does not name a file
// inside it. A problem with the file itself is reported as an outcome.
func (u Read) Execute(
	ctx context.Context, v domain.Vault, path string, start, length int,
) (_ ReadResult, err error) {
	length, err = checkRange(path, start, length)
	if err != nil {
		return ReadResult{}, err
	}
	out := ReadResult{Start: start}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return ReadResult{}, err
	}
	file, err := reader.Open(ctx, path)
	switch {
	case errors.Is(err, edit.ErrNotExist):
		out.Outcome = Missing
		return out, nil
	case err != nil:
		return ReadResult{}, err
	}
	defer closeAndJoin(&err, file)

	// A listing of the folder above is the vault saying its contents there.
	target, err := getReadOutcome(ctx, reader, path)
	if err != nil {
		return ReadResult{}, err
	}
	if target != Ok {
		out.Outcome = target
		return out, nil
	}

	return readRange(file, path, start, length)
}

// ExecuteFileRange reads length bytes of the file at path from start, with the file stat taken
// before the bytes are read.
func (u Read) ExecuteFileRange(
	ctx context.Context, v domain.Vault, path string, start, length int,
) (plugin.FileRange, error) {
	reader, err := u.Readers.Open(v)
	if err != nil {
		return plugin.FileRange{}, err
	}
	pos, err := reader.StatFile(ctx, path)
	if errors.Is(err, edit.ErrNotANote) {
		return plugin.FileRange{}, fmt.Errorf("%s is not a file in the vault: %w", path, edit.ErrNotExist)
	}
	if err != nil {
		return plugin.FileRange{}, err
	}
	if length <= 0 || length > MostRead {
		length = MostRead
	}
	result, err := u.Execute(ctx, v, path, start, length)
	if err != nil {
		return plugin.FileRange{}, err
	}
	switch result.Outcome {
	case Ok:
		return plugin.FileRange{Text: result.Text, Start: result.Start, Stat: pos}, nil
	case NotText:
		return plugin.FileRange{}, fmt.Errorf("read %s from %d: %w", path, result.Start, plugin.ErrNotText)
	case Missing, AFolder, LeftAlone:
	}
	return plugin.FileRange{}, fmt.Errorf("%s is %s: %w", path, result.Outcome, edit.ErrNotExist)
}

func checkRange(path string, start, length int) (int, error) {
	if start < 0 {
		return 0, fmt.Errorf("a range of %s begins at %d", path, start)
	}
	if length < 0 || length > MostRead {
		return 0, fmt.Errorf("read between 1 and %d bytes of a file at a time", MostRead)
	}
	if length == 0 {
		return MostRead, nil
	}
	return length, nil
}

func readRange(file io.ReadSeeker, path string, start, length int) (ReadResult, error) {
	var out ReadResult
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

	start, raw = text.WholeCharacters(start, raw, out.Whole)
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

func getReadOutcome(ctx context.Context, reader vaultport.VaultReader, path string) (ReadOutcome, error) {
	clean := pathpkg.Clean(strings.ReplaceAll(path, `\`, "/"))
	folder := pathpkg.Dir(clean)
	if folder == "." {
		folder = ""
	}
	entries, err := reader.List(ctx, folder)
	if errors.Is(err, edit.ErrNotExist) {
		return LeftAlone, nil
	}
	if err != nil {
		return "", err
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
