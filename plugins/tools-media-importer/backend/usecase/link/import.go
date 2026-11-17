package link

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

// ImportRequest is one address to import. An empty Vault is the vault the host shows.
type ImportRequest struct {
	Vault               domain.VaultID
	Folder              string
	URL                 string
	ShouldDownloadVideo bool
}

// ImportResult is the link note of an address and what was fetched for it. Error is the reason the
// text was not fetched, and DownloadError the reason no download was started.
type ImportResult struct {
	Path           string
	Title          string
	Producer       string
	Bytes          int
	HasText        bool
	Error          string
	DownloadTaskID string
	DownloadError  string
}

// ImportURL creates the link note of the address, or finds it, fetches its text before it returns,
// and starts the download of the video when requested. A failed fetch is reported in the result and
// the note is kept.
func (l *Links) ImportURL(ctx context.Context, req ImportRequest) (ImportResult, error) {
	made, _, err := l.FindOrCreate(ctx, req.Vault, req.Folder, req.URL)
	if err != nil {
		return ImportResult{}, err
	}
	out := ImportResult{
		Path: made.Path, Title: "", Producer: "", Bytes: 0, HasText: false, Error: "", DownloadTaskID: "", DownloadError: "",
	}
	if err := l.fetchImportedText(ctx, made, &out); err != nil {
		out.Error = err.Error()
	}
	if req.ShouldDownloadVideo {
		l.startImportedCopy(ctx, made, &out)
	}
	return out, nil
}

// CreateLinks creates the link notes of the addresses in folder and starts the fetch of each new one.
// It returns the paths of the notes, once each, and the addresses that failed; an address that is not
// a web address is left out of both.
func (l *Links) CreateLinks(ctx context.Context, folder string, urls []string) (paths, failed []string) {
	paths, failed = []string{}, []string{}
	for _, raw := range urls {
		made, isFound, err := l.FindOrCreate(ctx, "", folder, raw)
		if errors.Is(err, address.ErrNotAURL) {
			continue
		}
		if err != nil {
			failed = append(failed, raw)
			continue
		}
		if !isFound {
			l.StartFetch(ctx, made)
		}
		if !slices.Contains(paths, made.Path) {
			paths = append(paths, made.Path)
		}
	}
	return paths, failed
}

func (l *Links) fetchImportedText(ctx context.Context, made Link, out *ImportResult) error {
	fetchErr := l.FetchText(ctx, made)
	return errors.Join(fetchErr, l.readImportedText(ctx, made, out))
}

func (l *Links) readImportedText(ctx context.Context, made Link, out *ImportResult) error {
	text, err := l.ReadArticle(ctx, made)
	if err != nil || text == "" {
		return err
	}
	info, err := l.GetTextInfo(ctx, made)
	if err != nil {
		return err
	}
	out.Title, out.Producer, out.HasText, out.Bytes = info.Title, info.Producer, true, len(text)
	return nil
}

func (l *Links) startImportedCopy(ctx context.Context, made Link, out *ImportResult) {
	if !l.IsCopyable(made.Address) {
		out.DownloadError = ErrNotCopyable.Error()
		return
	}
	l.StartCopy(ctx, made, fetch.MediaVideo)
	out.DownloadTaskID = GetCopyTaskID(made.Vault, made.Path)
}
