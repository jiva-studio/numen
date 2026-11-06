// Package service implements the vault tools served to agents: reading and writing notes and files,
// listing links and headings, searching and describing the vault.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// FileReadInput is the input of the file_read tool.
type FileReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset to start reading from"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes to read; 0 or omitted reads to end of file"`
}

// FileReadOutput is the output of the file_read tool.
type FileReadOutput struct {
	Path        string `json:"path"`
	Content     string `json:"content" jsonschema:"the bytes read as text"`
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
			"not text — a PDF, a book, a recording — is refused: library_read reads its text.",
		Effect:       tools.ReadOnly,
		Servers:      []tools.Server{tools.Full, tools.Reading},
		Instructions: []string{fileReadSentence},
	}, inVault(t, (*Tools).readFile))
}

func (t *Tools) readFile(ctx context.Context, in FileReadInput) (FileReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return FileReadOutput{}, errors.New("path is required")
	}

	data, fp, err := t.vault.Read(ctx, in.Path)
	if err != nil {
		return FileReadOutput{}, fmt.Errorf("%s cannot be read: %w", in.Path, err)
	}
	if !isText(data) {
		return FileReadOutput{}, fmt.Errorf("%s is not a text file; library_read reads the text of a PDF, a book or a recording", in.Path)
	}

	start := in.Start
	if start < 0 {
		start = 0
	}

	var slice []byte
	if start >= len(data) {
		slice = []byte{}
	} else {
		end := len(data)
		if in.Length > 0 && start+in.Length < end {
			end = start + in.Length
		}
		slice = data[start:end]
	}

	return FileReadOutput{
		Path:        in.Path,
		Content:     string(slice),
		Start:       start,
		Length:      len(slice),
		Size:        len(data),
		Fingerprint: formatFingerprint(fp),
	}, nil
}

// FileCreateInput is the input of the file_create tool.
type FileCreateInput struct {
	Path    string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Content string `json:"content" jsonschema:"raw content to write to the new file"`
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

	data := []byte(in.Content)
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
