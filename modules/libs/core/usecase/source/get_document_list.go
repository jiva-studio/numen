package source

import (
	"context"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Document is a book, paper or scan filed in the vault, and whether a model has read it.
type Document struct {
	Path   string
	IsRead bool
}

// GetDocumentList lists the documents of a vault and says of each whether a model
// has read it.
type GetDocumentList struct {
	queries port.SourceQueries
}

func NewGetDocumentList(queries port.SourceQueries) *GetDocumentList {
	return &GetDocumentList{queries: queries}
}

func (u *GetDocumentList) Execute(ctx context.Context, v domain.Vault) ([]Document, error) {
	known, err := u.queries.GetFingerprints(ctx, v.ID, domain.KindBook)
	if err != nil {
		return nil, err
	}
	recognised, err := u.queries.GetRecognisedSources(ctx, v.ID, domain.KindBook)
	if err != nil {
		return nil, err
	}
	isRead := make(map[string]bool, len(recognised))
	for _, one := range recognised {
		isRead[one.Path] = one.Producer != ""
	}
	documents := make([]Document, 0, len(known))
	for path := range known {
		documents = append(documents, Document{Path: path, IsRead: isRead[path]})
	}
	slices.SortFunc(documents, func(a, b Document) int { return strings.Compare(a.Path, b.Path) })
	return documents, nil
}
