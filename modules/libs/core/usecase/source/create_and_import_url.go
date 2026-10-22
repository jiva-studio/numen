package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CreateAndImportURL makes the file a web address is kept in and fetches what is there.
type CreateAndImportURL struct {
	create *CreateURL
	fetch  *ImportURL
}

func NewCreateAndImportURL(create *CreateURL, fetch *ImportURL) *CreateAndImportURL {
	return &CreateAndImportURL{create: create, fetch: fetch}
}

// ImportedURL is the file that now exists and what was fetched for it, with the
// copy of the video when one was asked for.
type ImportedURL struct {
	ImportURLResult
	Copy CopyResult
}

// CopyError is a copy of the video that could not be made. The file and what was
// fetched for it stand.
type CopyError struct{ Err error }

func (e *CopyError) Error() string { return e.Err.Error() }

func (e *CopyError) Unwrap() error { return e.Err }

// Execute returns an error when the file could not be made or the address could not
// copy that fails comes back as a *CopyError beside the import it followed.
func (u *CreateAndImportURL) Execute(
	ctx context.Context, v domain.Vault, in NewURL, shouldCopy bool,
) (ImportedURL, error) {
	made, err := u.create.Execute(ctx, v, in)
	if err != nil {
		return ImportedURL{ImportURLResult: ImportURLResult{Path: made.Path}}, err
	}
	fetched, err := u.fetch.Execute(ctx, v, made.Path, false)
	if err != nil {
		return ImportedURL{ImportURLResult: ImportURLResult{Path: made.Path}}, err
	}
	res := ImportedURL{ImportURLResult: fetched}
	if !shouldCopy {
		return res, nil
	}
	res.Copy, err = u.fetch.Copy(ctx, v, fetched.Path, nil)
	if err != nil {
		return res, &CopyError{Err: err}
	}
	return res, nil
}
