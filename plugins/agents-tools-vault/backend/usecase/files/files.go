// Package files reads and creates raw files of a vault.
package files

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/fingerprint"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/ports"
)

// readMost is the maximum number of bytes one read returns, the maximum the host reads at once.
const readMost = 16000

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
func (u *UseCase) Read(ctx context.Context, in api.FileReadInput) (api.FileReadOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.FileReadOutput{}, errors.New("path is required")
	}
	length := in.Length
	if length <= 0 || length > readMost {
		length = readMost
	}
	read, err := u.reader.ReadRange(ctx, in.Path, max(in.Start, 0), length)
	if errors.Is(err, host.ErrNotText) {
		return api.FileReadOutput{}, fmt.Errorf("%s is not a text file; library_read reads the text of a PDF, a book or a recording", in.Path)
	}
	if err != nil {
		return api.FileReadOutput{}, fmt.Errorf("%s cannot be read: %w", in.Path, err)
	}
	return api.FileReadOutput{
		Path:        in.Path,
		Text:        read.Text,
		Start:       read.Start,
		Length:      len(read.Text),
		Size:        int(read.Stat.Size),
		Fingerprint: fingerprint.Format(read.Stat),
	}, nil
}

// Create writes a new file and updates the index for it.
func (u *UseCase) Create(ctx context.Context, in api.FileCreateInput) (api.FileCreateOutput, error) {
	if strings.TrimSpace(in.Path) == "" {
		return api.FileCreateOutput{}, errors.New("path is required")
	}
	data := []byte(in.Text)
	if err := u.writer.Create(ctx, in.Path, data); err != nil {
		return api.FileCreateOutput{}, fmt.Errorf("%s cannot be created: %w", in.Path, err)
	}
	if err := u.index.Scan(ctx, []string{in.Path}); err != nil {
		return api.FileCreateOutput{}, fmt.Errorf("%s cannot be indexed: %w", in.Path, err)
	}
	return api.FileCreateOutput{Path: in.Path, Size: len(data)}, nil
}
