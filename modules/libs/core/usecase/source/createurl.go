package source

import (
	"context"
	pathpkg "path"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/urlfile"
)

// CreateURL makes the file a web address is kept in.
//
// It holds the address and nothing else, so what a person writes about what is
// there is a note of their own, pointing at this file.
type CreateURL struct {
	Writers port.VaultWriters

	// Index brings the file into the index, so it is searched and openable as
	// soon as it is made. A run given none leaves that to the next walk.
	Index func(ctx context.Context, v domain.Vault, paths []string) error
}

// NewURL is one address a caller wants a file made for.
type NewURL struct {
	Address domain.URL
	// Title is what the file is called. Empty is the address itself, which is
	// the name a download replaces once it knows what is there.
	Title string
	// Path is the folder it goes in, relative to the root. Empty is the root
	// itself.
	Path string
}

// CreateURLResult is the file that now exists.
type CreateURLResult struct {
	Path  string
	Title string
}

func (u CreateURL) Execute(
	ctx context.Context, v domain.Vault, in NewURL,
) (CreateURLResult, error) {
	title := in.Title
	if title == "" {
		title = string(in.Address)
	}
	name, _, err := domain.Filename(title)
	if err != nil {
		return CreateURLResult{}, err
	}
	path := pathpkg.Join(in.Path, name+domain.URLExtension)

	writer, err := u.Writers.Open(v)
	if err != nil {
		return CreateURLResult{}, err
	}
	// Whether the path was free is the filesystem's to answer, at the moment
	// the file is made.
	if err := writer.Create(ctx, path, urlfile.Write(in.Address)); err != nil {
		return CreateURLResult{}, err
	}

	made := CreateURLResult{Path: path, Title: title}
	if u.Index == nil {
		return made, nil
	}
	return made, u.Index(ctx, v, []string{path})
}
