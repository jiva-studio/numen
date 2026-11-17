// Package notes reads, writes, lists and searches the notes of a vault.
package notes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// Note is a note with its title, body, frontmatter and fingerprint.
type Note struct {
	Path        string
	Title       string
	Text        string
	Body        string
	Fingerprint string
	Frontmatter map[string]any
}

// Written is the note a write wrote, its fingerprint after the write and the bytes written.
type Written struct {
	Path        string
	Fingerprint string
	Bytes       int
}

// Heading is a heading of a note; Line counts from 1 at the first line of the body and Start is the
// byte offset in the body.
type Heading struct {
	Level int
	Text  string
	Line  int
	Start int
}

// UseCase works the notes of one vault.
type UseCase struct {
	reader   ports.FileReader
	writer   ports.FileWriter
	index    ports.Indexer
	lookup   ports.NoteLookup
	resolver ports.NotePathResolver
}

// New creates a UseCase over the notes of one vault; resolver finds the note an agent names.
func New(reader ports.FileReader, writer ports.FileWriter, index ports.Indexer, lookup ports.NoteLookup, resolver ports.NotePathResolver) *UseCase {
	return &UseCase{reader: reader, writer: writer, index: index, lookup: lookup, resolver: resolver}
}

// Read returns a note with its title, body, frontmatter and fingerprint.
func (u *UseCase) Read(ctx context.Context, name string) (Note, error) {
	if strings.TrimSpace(name) == "" {
		return Note{}, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, name)
	data, stat, err := u.reader.Read(ctx, path)
	if err != nil {
		return Note{}, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	parsed := markdown.Parse(stat, data)
	return Note{
		Path:        path,
		Title:       parsed.Title,
		Text:        string(data),
		Body:        parsed.Body,
		Fingerprint: fingerprint.Format(stat),
		Frontmatter: parsed.Frontmatter,
	}, nil
}

// Write replaces a note, or creates it when it does not exist and no fingerprint is given. text is nil
// when the caller gave none.
func (u *UseCase) Write(ctx context.Context, path string, text *string, token string) (Written, error) {
	if strings.TrimSpace(path) == "" {
		return Written{}, errors.New("path is required")
	}
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	if text == nil {
		return Written{}, errors.New("text is required")
	}
	content := *text
	stat, err := u.save(ctx, path, content, token)
	if err != nil {
		return Written{}, err
	}
	if err := u.index.Scan(ctx, []string{path}); err != nil {
		return Written{}, fmt.Errorf("%s cannot be indexed: %w", path, err)
	}
	return Written{Path: path, Fingerprint: fingerprint.Format(stat), Bytes: len(content)}, nil
}

func (u *UseCase) save(ctx context.Context, path, content, token string) (domain.FileStat, error) {
	if token != "" {
		return u.writeAgainst(ctx, path, content, token)
	}
	return u.writeOrCreate(ctx, path, content)
}

func (u *UseCase) writeAgainst(ctx context.Context, path, content, token string) (domain.FileStat, error) {
	baseline, err := fingerprint.Parse(token)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := u.writer.Write(ctx, path, []byte(content), baseline)
	if errors.Is(err, host.ErrStale) {
		return domain.FileStat{}, fmt.Errorf("%s changed since note_read returned the fingerprint; read it again: %w", path, err)
	}
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s cannot be written: %w", path, err)
	}
	return stat, nil
}

func (u *UseCase) writeOrCreate(ctx context.Context, path, content string) (domain.FileStat, error) {
	stat, err := u.writer.Write(ctx, path, []byte(content), domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}})
	if err == nil {
		return stat, nil
	}
	if !errors.Is(err, host.ErrNotExist) {
		return domain.FileStat{}, fmt.Errorf("%s cannot be written: %w", path, err)
	}
	if err := u.writer.Create(ctx, path, []byte(content)); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s cannot be created: %w", path, err)
	}
	stat, err = u.reader.StatFile(ctx, path)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	return stat, nil
}

// ListHeadings returns the path of the note an agent names and its headings.
func (u *UseCase) ListHeadings(ctx context.Context, name string) (string, []Heading, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, name)
	data, stat, err := u.reader.Read(ctx, path)
	if err != nil {
		return "", nil, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	parsed := markdown.Parse(stat, data)
	headings := make([]Heading, 0, len(parsed.Headings))
	for _, heading := range parsed.Headings {
		headings = append(headings, Heading{Level: heading.Level, Text: heading.Text, Line: heading.Line + 1, Start: heading.Offset})
	}
	return path, headings, nil
}
