// Package service implements the vault tools served to agents: reading and writing notes and files,
// listing links and headings, searching and describing the vault.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// fileReadMost is the maximum number of bytes one file_read returns, the maximum the host reads at once.
const fileReadMost = 16000

// FileReadInput is the input of the file_read tool.
type FileReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset to start reading from"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes to read, at most 16000; 16000 when left out"`
}

// FileReadOutput is the output of the file_read tool.
type FileReadOutput struct {
	Path        string `json:"path"`
	Text        string `json:"text" jsonschema:"the bytes read as text"`
	Start       int    `json:"start" jsonschema:"starting byte offset of the returned slice"`
	Length      int    `json:"length" jsonschema:"number of bytes in the returned slice"`
	Size        int    `json:"size" jsonschema:"total size of the file in bytes"`
	Fingerprint string `json:"fingerprint" jsonschema:"file fingerprint"`
}

func (t *Tools) addFileRead(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "file_read",
		Title: "Read File",
		Description: "Read the content or a byte range of a text file of the vault, as text. A file that is " +
			"not text — a PDF, a book, a recording — returns an error; library_read reads its text.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{fileReadSentence},
	}, inVault(t, (*Tools).readFile))
}

func (t *Tools) readFile(ctx context.Context, in FileReadInput) (FileReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return FileReadOutput{}, errors.New("path is required")
	}
	length := in.Length
	if length <= 0 || length > fileReadMost {
		length = fileReadMost
	}
	read, err := t.vault.ReadRange(ctx, in.Path, max(in.Start, 0), length)
	if errors.Is(err, host.ErrNotText) {
		return FileReadOutput{}, fmt.Errorf("%s is not a text file; library_read reads the text of a PDF, a book or a recording", in.Path)
	}
	if err != nil {
		return FileReadOutput{}, fmt.Errorf("%s cannot be read: %w", in.Path, err)
	}
	return FileReadOutput{
		Path:        in.Path,
		Text:        read.Text,
		Start:       read.Start,
		Length:      len(read.Text),
		Size:        int(read.Stat.Size),
		Fingerprint: formatFingerprint(read.Stat),
	}, nil
}

// FileCreateInput is the input of the file_create tool.
type FileCreateInput struct {
	Path string  `json:"path" jsonschema:"the file's path relative to the vault root"`
	Text *string `json:"text,omitempty" jsonschema:"raw content to write to the new file"`
}

// FileCreateOutput is the output of the file_create tool.
type FileCreateOutput struct {
	Path string `json:"path" jsonschema:"the path of the created file, relative to the vault root"`
	Size int    `json:"size" jsonschema:"the size of the created file in bytes"`
}

func (t *Tools) addFileCreate(registry *tools.Registry) error {
	return addTool(registry, tools.Spec{
		Step:        tools.ByEffect,
		Name:        "file_create",
		Title:       "Create File",
		Description: "Create a new raw file in the vault with the specified content.",
		Effect:      tools.ReadWrite,
		Servers:     []tools.Server{tools.Full},
	}, inVault(t, (*Tools).createFile))
}

func (t *Tools) createFile(ctx context.Context, in FileCreateInput) (FileCreateOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return FileCreateOutput{}, errors.New("path is required")
	}

	if in.Text == nil {
		return FileCreateOutput{}, errors.New("text is required")
	}
	data := []byte(*in.Text)
	if err := t.vault.Create(ctx, in.Path, data); err != nil {
		return FileCreateOutput{}, fmt.Errorf("%s cannot be created: %w", in.Path, err)
	}

	if err := t.vault.Scan(ctx, []string{in.Path}); err != nil {
		return FileCreateOutput{}, fmt.Errorf("%s cannot be indexed: %w", in.Path, err)
	}

	return FileCreateOutput{
		Path: in.Path,
		Size: len(data),
	}, nil
}
