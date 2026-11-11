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
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/notepath"
)

// UseCase works the notes of one vault.
type UseCase struct {
	reader   ports.FileReader
	writer   ports.FileWriter
	index    ports.Indexer
	lookup   ports.NoteLookup
	resolver *notepath.Resolver
}

// New creates a UseCase over the notes of one vault.
func New(reader ports.FileReader, writer ports.FileWriter, index ports.Indexer, lookup ports.NoteLookup) *UseCase {
	return &UseCase{reader: reader, writer: writer, index: index, lookup: lookup, resolver: notepath.NewResolver(lookup, reader)}
}

// Read returns a note with its title, body, frontmatter and fingerprint.
func (u *UseCase) Read(ctx context.Context, in api.NoteReadInput) (api.NoteReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.NoteReadOutput{}, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, in.Path)
	data, stat, err := u.reader.Read(ctx, path)
	if err != nil {
		return api.NoteReadOutput{}, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	parsed := markdown.Parse(stat, data)
	return api.NoteReadOutput{
		Path:        path,
		Title:       parsed.Title,
		Text:        string(data),
		Body:        parsed.Body,
		Fingerprint: fingerprint.Format(stat),
		Frontmatter: parsed.Frontmatter,
	}, nil
}

// Write replaces a note, or creates it when it does not exist and no fingerprint is given.
func (u *UseCase) Write(ctx context.Context, in api.NoteWriteInput) (api.NoteWriteOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.NoteWriteOutput{}, errors.New("path is required")
	}
	path := in.Path
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	if in.Text == nil {
		return api.NoteWriteOutput{}, errors.New("text is required")
	}
	content := *in.Text
	stat, err := u.save(ctx, path, content, in.Fingerprint)
	if err != nil {
		return api.NoteWriteOutput{}, err
	}
	if err := u.index.Scan(ctx, []string{path}); err != nil {
		return api.NoteWriteOutput{}, fmt.Errorf("%s cannot be indexed: %w", path, err)
	}
	return api.NoteWriteOutput{Path: path, Fingerprint: fingerprint.Format(stat), Bytes: len(content)}, nil
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

// ListHeadings returns the headings of a note.
func (u *UseCase) ListHeadings(ctx context.Context, in api.NoteHeadingListInput) (api.NoteHeadingListOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.NoteHeadingListOutput{}, errors.New("path is required")
	}
	path := u.resolver.Resolve(ctx, in.Path)
	data, stat, err := u.reader.Read(ctx, path)
	if err != nil {
		return api.NoteHeadingListOutput{}, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	parsed := markdown.Parse(stat, data)
	items := make([]api.HeadingItem, 0, len(parsed.Headings))
	for _, heading := range parsed.Headings {
		items = append(items, api.HeadingItem{Level: heading.Level, Text: heading.Text, Line: heading.Line + 1, Start: heading.Offset})
	}
	return api.NoteHeadingListOutput{Path: path, Headings: items}, nil
}
