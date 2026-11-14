// Package editpreview proposes an edit of a note without writing the note.
package editpreview

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/diff"
)

var (
	errMissingPath = errors.New("path is required")
	errMissingEdit = errors.New("either proposedText or diff is required")
)

// Vault reads the note the edit is proposed to.
type Vault interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
}

// Files returns the reader of the vault with the identity; the empty identity is the vault the host
// shows. A nil Files means the host offers none.
type Files func(vault domain.VaultID) Vault

// Request is a proposed edit of a note in Vault, the vault the host shows when it is empty. Diff is a
// unified diff to show as it is; otherwise the diff is built from OriginalText, or the text of the
// note when that is empty, and ProposedText.
type Request struct {
	Vault        domain.VaultID
	Path         string
	ProposedText string
	OriginalText string
	Diff         string
}

// Preview is the diff of a proposed edit.
type Preview struct {
	Path      string
	Diff      string
	Additions int
	Deletions int
}

// Creator creates previews.
type Creator struct {
	files Files
}

// NewCreator creates a creator that reads notes through files; files may be nil.
func NewCreator(files Files) Creator {
	return Creator{files: files}
}

func (c Creator) Create(ctx context.Context, req Request) (Preview, error) {
	if req.Path == "" {
		return Preview{}, errMissingPath
	}
	if req.ProposedText == "" && req.Diff == "" {
		return Preview{}, errMissingEdit
	}
	diffText, additions, deletions := c.prepareDiff(ctx, req)
	return Preview{Path: req.Path, Diff: diffText, Additions: additions, Deletions: deletions}, nil
}

func (c Creator) prepareDiff(ctx context.Context, req Request) (string, int, int) {
	if req.Diff != "" {
		additions, deletions := diff.CountChanges(req.Diff)
		return req.Diff, additions, deletions
	}
	original := req.OriginalText
	if original == "" && c.files != nil {
		if vault := c.files(req.Vault); vault != nil {
			if data, _, err := vault.Read(ctx, req.Path); err == nil {
				original = string(data)
			}
		}
	}
	return diff.CreateUnified(req.Path, original, req.ProposedText)
}
