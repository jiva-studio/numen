package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/tools"
)

type FileReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset to start reading from"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes to read; 0 or omitted reads to end of file"`
}

type FileReadOutput struct {
	Path        string `json:"path"`
	Content     string `json:"content" jsonschema:"the bytes read as text"`
	Start       int    `json:"start" jsonschema:"starting byte offset of the returned slice"`
	Length      int    `json:"length" jsonschema:"number of bytes in the returned slice"`
	Size        int    `json:"size" jsonschema:"total size of the file in bytes"`
	Fingerprint string `json:"fingerprint" jsonschema:"file fingerprint"`
}

func (t *Tools) addFileRead(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "file_read",
		Title:       "Read File",
		Description: "Read raw content or a byte range of any file in the vault.",
		Effect:      tools.Reads,
		Servers:     []tools.Server{tools.Full, tools.Reading},
	}, t.readFile)
}

func (t *Tools) readFile(ctx context.Context, in FileReadInput) (FileReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return FileReadOutput{}, errors.New("path required")
	}

	data, fp, err := t.vault.Read(ctx, in.Path)
	if err != nil {
		return FileReadOutput{}, fmt.Errorf("read file %s: %w", in.Path, err)
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

type FileCreateInput struct {
	Path    string `json:"path" jsonschema:"the file's path relative to the vault root"`
	Content string `json:"content" jsonschema:"raw content to write to the new file"`
}

type FileCreateOutput struct {
	Path    string `json:"path"`
	Size    int    `json:"size"`
	Success bool   `json:"success"`
}

func (t *Tools) addFileCreate(registry *tools.Registry) error {
	return tools.Add(registry, tools.Spec{
		Name:        "file_create",
		Title:       "Create File",
		Description: "Create a new raw file in the vault with the specified content.",
		Effect:      tools.Writes,
		Servers:     []tools.Server{tools.Full},
	}, t.createFile)
}

func (t *Tools) createFile(ctx context.Context, in FileCreateInput) (FileCreateOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return FileCreateOutput{}, errors.New("path required")
	}

	data := []byte(in.Content)
	if err := t.vault.Create(ctx, in.Path, data); err != nil {
		return FileCreateOutput{}, fmt.Errorf("create file %s: %w", in.Path, err)
	}

	_ = t.vault.Scan(ctx, []string{in.Path})

	return FileCreateOutput{
		Path:    in.Path,
		Size:    len(data),
		Success: true,
	}, nil
}
