// Package editpreview proposes an edit of a note without writing the note.
package editpreview

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/diff"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/usecase/preview"
)

var (
	errMissingPath = errors.New("path is required")
	errMissingEdit = errors.New("either proposedText or diff is required")
)

// Vault reads the note the edit is proposed to. A nil Vault means the host offers none.
type Vault interface {
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
}

// Request is a proposed edit. Diff is a unified diff to show as it is; otherwise the diff is
// built from OriginalText, or the text of the note when that is empty, and ProposedText.
type Request struct {
	Path         string
	ProposedText string
	OriginalText string
	Diff         string
}

// Creator creates previews and keeps them in a store.
type Creator struct {
	vault Vault
	store *preview.Store
}

// NewCreator creates a creator that keeps its previews in store; vault may be nil.
func NewCreator(store *preview.Store, vault Vault) Creator {
	return Creator{vault: vault, store: store}
}

func (c Creator) Create(ctx context.Context, req Request) (preview.Preview, error) {
	if req.Path == "" {
		return preview.Preview{}, errMissingPath
	}
	if req.ProposedText == "" && req.Diff == "" {
		return preview.Preview{}, errMissingEdit
	}
	diffText, additions, deletions := c.prepareDiff(ctx, req)
	created := preview.Preview{
		ID:           c.store.NextID(),
		Path:         req.Path,
		ProposedText: req.ProposedText,
		OriginalText: req.OriginalText,
		Diff:         diffText,
		Additions:    additions,
		Deletions:    deletions,
		CreatedAt:    c.store.Now(),
	}
	c.store.Save(created)
	return created, nil
}

func (c Creator) prepareDiff(ctx context.Context, req Request) (string, int, int) {
	if req.Diff != "" {
		additions, deletions := diff.CountChanges(req.Diff)
		return req.Diff, additions, deletions
	}
	original := req.OriginalText
	if original == "" && c.vault != nil {
		if data, _, err := c.vault.Read(ctx, req.Path); err == nil {
			original = string(data)
		}
	}
	return diff.CreateUnified(req.Path, original, req.ProposedText)
}
