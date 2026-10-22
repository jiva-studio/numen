package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
)

type NoteReadInput struct {
	Path string `json:"path" jsonschema:"relative path to the note file (e.g. Notes/Topic.md)"`
}

type NoteReadOutput struct {
	Path        string         `json:"path" jsonschema:"path of the note"`
	Title       string         `json:"title" jsonschema:"title of the note"`
	Content     string         `json:"content" jsonschema:"full raw markdown content of the file"`
	Body        string         `json:"body" jsonschema:"prose content of the note excluding frontmatter"`
	Fingerprint string         `json:"fingerprint" jsonschema:"concurrency fingerprint for atomic writes"`
	Frontmatter map[string]any `json:"frontmatter,omitempty" jsonschema:"parsed YAML frontmatter, if present"`
}

func (t *Tools) addNoteRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "note_read",
		Title:       "Read Note",
		Description: "Read the content, body prose, title, and metadata of a note in the vault.",
		Effect:      tools.Reads,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, t.readNote)
}

func (t *Tools) resolveNotePath(ctx context.Context, p string) string {
	if strings.TrimSpace(p) == "" {
		return p
	}
	p = strings.TrimSpace(p)
	if strings.HasSuffix(p, ".md") {
		return p
	}
	named, err := t.vault.GetNamedPaths(ctx, p)
	if err == nil && len(named) > 0 {
		return named[0]
	}
	if _, err := t.vault.GetFingerprint(ctx, p+".md"); err == nil {
		return p + ".md"
	}
	return p
}

func (t *Tools) readNote(ctx context.Context, in NoteReadInput) (NoteReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return NoteReadOutput{}, errors.New("path required")
	}
	path := t.resolveNotePath(ctx, in.Path)
	data, fp, err := t.vault.Read(ctx, path)
	if err != nil {
		return NoteReadOutput{}, fmt.Errorf("read note %s: %w", path, err)
	}

	parsed := markdown.Parse(fp, data)
	return NoteReadOutput{
		Path:        path,
		Title:       parsed.Title,
		Content:     string(data),
		Body:        parsed.Body,
		Fingerprint: formatFingerprint(fp),
		Frontmatter: parsed.Frontmatter,
	}, nil
}

type NoteWriteInput struct {
	Path        string `json:"path" jsonschema:"relative path to the note file"`
	Content     string `json:"content,omitempty" jsonschema:"markdown content to write"`
	Body        string `json:"body,omitempty" jsonschema:"alias for content: markdown text to write"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"optional baseline fingerprint from note_read for optimistic concurrency"`
}

type NoteWriteOutput struct {
	Path        string `json:"path" jsonschema:"path written"`
	Fingerprint string `json:"fingerprint" jsonschema:"new concurrency fingerprint after write"`
	Bytes       int    `json:"bytes" jsonschema:"number of bytes written"`
}

func (t *Tools) addNoteWrite(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "note_write",
		Title:       "Write Note",
		Description: "Atomically write or create a note in the vault. If fingerprint is provided, verifies optimistic concurrency.",
		Effect:      tools.Writes,
		Servers:     []tools.Server{tools.Full},
	}, t.writeNote)
}

func (t *Tools) writeNote(ctx context.Context, in NoteWriteInput) (NoteWriteOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return NoteWriteOutput{}, errors.New("path required")
	}
	path := in.Path
	if !strings.HasSuffix(path, ".md") {
		path = path + ".md"
	}

	content := in.Content
	if content == "" && in.Body != "" {
		content = in.Body
	}

	newFp, err := t.saveNoteContent(ctx, path, content, in.Fingerprint)
	if err != nil {
		return NoteWriteOutput{}, err
	}

	_ = t.vault.Scan(ctx, []string{path})

	return NoteWriteOutput{
		Path:        path,
		Fingerprint: formatFingerprint(newFp),
		Bytes:       len(content),
	}, nil
}

func (t *Tools) saveNoteContent(ctx context.Context, path, content, fpStr string) (domain.Fingerprint, error) {
	if fpStr != "" {
		return t.writeWithFingerprint(ctx, path, content, fpStr)
	}
	return t.writeOrCreate(ctx, path, content)
}

func (t *Tools) writeWithFingerprint(ctx context.Context, path, content, fpStr string) (domain.Fingerprint, error) {
	baseline, err := parseFingerprint(fpStr)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	newFp, err := t.vault.Write(ctx, path, []byte(content), baseline)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("write note %s: %w", path, err)
	}
	return newFp, nil
}

func (t *Tools) writeOrCreate(ctx context.Context, path, content string) (domain.Fingerprint, error) {
	writtenFp, err := t.vault.Write(ctx, path, []byte(content), domain.Fingerprint{})
	if err == nil {
		return writtenFp, nil
	}
	if !errors.Is(err, host.ErrNotExist) {
		return domain.Fingerprint{}, fmt.Errorf("write note %s: %w", path, err)
	}
	if err := t.vault.Create(ctx, path, []byte(content)); err != nil {
		return domain.Fingerprint{}, fmt.Errorf("create note %s: %w", path, err)
	}
	newFp, _ := t.vault.GetFingerprint(ctx, path)
	return newFp, nil
}

type HeadingItem struct {
	Level  int    `json:"level" jsonschema:"heading level (1 to 6)"`
	Text   string `json:"text" jsonschema:"heading text"`
	Line   int    `json:"line" jsonschema:"line index in the body"`
	Offset int    `json:"offset" jsonschema:"byte offset in the body"`
}

type NoteHeadingsInput struct {
	Path string `json:"path" jsonschema:"relative path to the note file"`
}

type NoteHeadingsOutput struct {
	Path     string        `json:"path"`
	Headings []HeadingItem `json:"headings"`
}

func (t *Tools) addNoteHeadings(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "note_headings",
		Title:       "Extract Note Headings",
		Description: "Extract the outline and headings from a note in the vault.",
		Effect:      tools.Reads,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, t.getNoteHeadings)
}

func (t *Tools) getNoteHeadings(ctx context.Context, in NoteHeadingsInput) (NoteHeadingsOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return NoteHeadingsOutput{}, errors.New("path required")
	}
	path := t.resolveNotePath(ctx, in.Path)
	data, fp, err := t.vault.Read(ctx, path)
	if err != nil {
		return NoteHeadingsOutput{}, fmt.Errorf("read note %s: %w", path, err)
	}

	parsed := markdown.Parse(fp, data)
	items := make([]HeadingItem, 0, len(parsed.Headings))
	for _, h := range parsed.Headings {
		items = append(items, HeadingItem{
			Level:  h.Level,
			Text:   h.Text,
			Line:   h.Line,
			Offset: h.Offset,
		})
	}

	return NoteHeadingsOutput{
		Path:     path,
		Headings: items,
	}, nil
}
