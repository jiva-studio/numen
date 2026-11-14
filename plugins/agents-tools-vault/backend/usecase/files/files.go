// Package files reads and creates raw files of a vault.
package files

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// readMost is the maximum number of bytes one read returns, the maximum the host reads at once.
const readMost = 16000

// FileText is a range of the text of a file.
type FileText struct {
	Path        string
	Text        string
	Start       int
	Size        int
	Fingerprint string
}

// UseCase reads and creates the files of one vault.
type UseCase struct {
	reader ports.FileReader
	writer ports.FileWriter
	index  ports.Indexer
}

// New creates a UseCase over the files of one vault.
func New(reader ports.FileReader, writer ports.FileWriter, index ports.Indexer) *UseCase {
	return &UseCase{reader: reader, writer: writer, index: index}
}

// Read returns the text of a range of a text file.
func (u *UseCase) Read(ctx context.Context, path string, start, length int) (FileText, error) {
	if strings.TrimSpace(path) == "" {
		return FileText{}, errors.New("path is required")
	}
	if length <= 0 || length > readMost {
		length = readMost
	}
	read, err := u.reader.ReadRange(ctx, path, max(start, 0), length)
	if errors.Is(err, host.ErrNotText) {
		return FileText{}, fmt.Errorf("%s is not a text file; library_read reads the text of a PDF, a book or a recording", path)
	}
	if err != nil {
		return FileText{}, fmt.Errorf("%s cannot be read: %w", path, err)
	}
	return FileText{
		Path:        path,
		Text:        read.Text,
		Start:       read.Start,
		Size:        int(read.Stat.Size),
		Fingerprint: fingerprint.Format(read.Stat),
	}, nil
}

// Create writes a new file, updates the index for it and returns its size in bytes.
func (u *UseCase) Create(ctx context.Context, path, text string) (int, error) {
	if strings.TrimSpace(path) == "" {
		return 0, errors.New("path is required")
	}
	data := []byte(text)
	if err := u.writer.Create(ctx, path, data); err != nil {
		return 0, fmt.Errorf("%s cannot be created: %w", path, err)
	}
	if err := u.index.Scan(ctx, []string{path}); err != nil {
		return 0, fmt.Errorf("%s cannot be indexed: %w", path, err)
	}
	return len(data), nil
}
