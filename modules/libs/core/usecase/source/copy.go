package source

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// Downloading the media a url points at, so a person plays it from this disk.

// CopyResult reports what downloading a copy did.
type CopyResult struct {
	Path string
	// Bytes is how large what is at the address is, and Limit the size a copy
	// may be. Over the limit nothing is fetched.
	Bytes int64
	Limit int64
	// Existed is a copy that already stood, so nothing was fetched.
	Existed bool
	// At is where in the vault the copy landed, and nothing where it landed in
	// the application's own folder.
	At string
}

// TooLarge is a copy the limit refused. Nothing was fetched.
func (r CopyResult) TooLarge() bool { return r.Limit > 0 && r.Bytes > r.Limit }

// Copy fetches what is at a url's address so a person plays it from this disk.
//
// It is asked for by hand: a copy takes as much of somebody's disk as what is
// at the address takes, and pasting an address is not asking for one. Where it
// lands is `importing.copies_to_vault`: the application's own folder, where
// losing it costs another fetch, or beside the url as a file of the person's
// own.
func (u ImportURL) Copy(ctx context.Context, v domain.Vault, path string) (CopyResult, error) {
	res := CopyResult{Path: path, Limit: u.CopyMaxSize}
	at, _, store, err := u.pointed(ctx, v, path)
	if err != nil {
		return res, err
	}
	hash := text.Fingerprint([]byte(string(at)))
	beside := CopyBeside(path)

	// One run to a copy, held for as long as the fetch takes. The claim is on
	// the store's name whichever disk the copy lands on: it is the address that
	// is being fetched, and two urls on one address are one fetch.
	release, err := store.Claim(ctx, text.Copy(hash))
	if errors.Is(err, port.ErrClaimed) {
		return res, ErrBeingFetched
	}
	if err != nil {
		return res, err
	}
	defer release()

	if size, held, err := u.stands(ctx, v, store, hash, beside); err != nil {
		return res, err
	} else if held {
		res.Existed, res.Bytes, res.At = true, size, u.landing(beside)
		return res, nil
	}

	meta, err := u.By.Metadata(ctx, at)
	if err != nil {
		return res, err
	}
	res.Bytes = meta.Bytes
	if u.CopyMaxSize > 0 && meta.Bytes > u.CopyMaxSize {
		return res, nil
	}

	// The copy is written through the store's own rename, so a fetch that
	// stopped leaves nothing anything plays.
	read, write := io.Pipe()
	going := make(chan error, 1)
	go func() {
		_, err := u.By.Download(ctx, at, write)
		_ = write.CloseWithError(err)
		going <- err
	}()
	arriving := capped(read, u.CopyMaxSize)
	if u.Progress != nil {
		u.Progress(0, meta.Bytes)
		arriving = &passing{from: arriving, total: meta.Bytes, report: u.Progress}
	}
	size, err := u.takes(ctx, v, store, text.Copy(hash), beside, arriving)
	// The read end is closed before the fetch is waited for: whoever stopped
	// reading unblocks whoever is writing into it.
	_ = read.CloseWithError(err)
	fetching := <-going
	// A site that declared a size it then ran past is known only to be over the
	// limit, which is the size reported.
	if errors.Is(err, errTooLarge) {
		res.Bytes = u.CopyMaxSize + 1
		return res, nil
	}
	if fetching != nil {
		return res, fetching
	}
	if err != nil {
		return res, err
	}
	res.Bytes, res.At = size, u.landing(beside)
	return res, nil
}

// CopyBeside is where a copy kept in the vault stands: beside the url, under
// its own name. A person looking at the folder sees the copy and the url
// together.
func CopyBeside(path string) string {
	return strings.TrimSuffix(path, domain.URLExtension) + text.CopyExtension
}

// landing is where the copy is, as the vault names it, and nothing where it is
// in the application's own folder.
func (u ImportURL) landing(beside string) string {
	if !u.ToVault {
		return ""
	}
	return beside
}

// stands says whether a copy is already here, and how large.
func (u ImportURL) stands(
	ctx context.Context, v domain.Vault, store port.DerivedStore, hash, beside string,
) (int64, bool, error) {
	if u.ToVault {
		reader, err := u.Readers.Open(v)
		if err != nil {
			return 0, false, err
		}
		ref, err := reader.Stat(ctx, beside)
		if errors.Is(err, fs.ErrNotExist) {
			return 0, false, nil
		}
		if err != nil {
			return 0, false, err
		}
		return ref.Size, true, nil
	}
	_, size, err := store.Open(ctx, text.Copy(hash))
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return size, true, nil
}

// takes writes the copy where the settings say it lands, and answers how large
// it came to.
//
// A copy in the vault is the person's file: the application does not replace
// one that is there, and putting the folders above it right is the writer's.
func (u ImportURL) takes(
	ctx context.Context, v domain.Vault, store port.DerivedStore, name, beside string, from io.Reader,
) (int64, error) {
	if !u.ToVault {
		return store.Take(ctx, name, from)
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return 0, err
	}
	counted := &passing{from: from}
	if err := writer.Bring(ctx, beside, counted); err != nil {
		return 0, err
	}
	return counted.read, nil
}
