// Package linkimport imports addresses as link notes and starts the fetches of their text and media.
package linkimport

import (
	"context"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// Importer imports addresses over the links of the vault.
type Importer struct {
	links *link.Links
}

// New returns an importer over links.
func New(links *link.Links) *Importer {
	return &Importer{links: links}
}

// Request is one address to import. An empty Vault is the vault the host shows.
type Request struct {
	Vault               domain.VaultID
	Folder              string
	URL                 string
	ShouldDownloadVideo bool
}

// Result is the link note of an address and what was fetched for it. Error is the reason the text was
// not fetched, and DownloadError the reason no download was started.
type Result struct {
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
func (i *Importer) ImportURL(ctx context.Context, req Request) (Result, error) {
	made, _, err := i.links.FindOrCreate(ctx, req.Vault, req.Folder, req.URL)
	if err != nil {
		return Result{}, err
	}
	out := Result{
		Path: made.Path, Title: "", Producer: "", Bytes: 0, HasText: false, Error: "", DownloadTaskID: "", DownloadError: "",
	}
	if err := i.fetchText(ctx, made, &out); err != nil {
		out.Error = err.Error()
	}
	if req.ShouldDownloadVideo {
		i.startDownload(ctx, made, &out)
	}
	return out, nil
}

// CreateLinks creates the link notes of the addresses in folder and starts the fetch of each new one.
// It returns the paths of the notes, once each, and the addresses that failed; an address that is not
// a web address is left out of both.
func (i *Importer) CreateLinks(ctx context.Context, folder string, urls []string) (paths, failed []string) {
	paths, failed = []string{}, []string{}
	for _, raw := range urls {
		made, isFound, err := i.links.FindOrCreate(ctx, "", folder, raw)
		if errors.Is(err, address.ErrNotAURL) {
			continue
		}
		if err != nil {
			failed = append(failed, raw)
			continue
		}
		if !isFound {
			i.links.StartFetch(ctx, made)
		}
		if !slices.Contains(paths, made.Path) {
			paths = append(paths, made.Path)
		}
	}
	return paths, failed
}

func (i *Importer) fetchText(ctx context.Context, made link.Link, out *Result) error {
	fetchErr := i.links.FetchText(ctx, made)
	return errors.Join(fetchErr, i.readText(ctx, made, out))
}

func (i *Importer) readText(ctx context.Context, made link.Link, out *Result) error {
	text, err := i.links.ReadArticle(ctx, made)
	if err != nil || text == "" {
		return err
	}
	info, err := i.links.GetTextInfo(ctx, made)
	if err != nil {
		return err
	}
	out.Title, out.Producer, out.HasText, out.Bytes = info.Title, info.Producer, true, len(text)
	return nil
}

func (i *Importer) startDownload(ctx context.Context, made link.Link, out *Result) {
	if !i.links.IsCopyable(made.Address) {
		out.DownloadError = link.ErrNotCopyable.Error()
		return
	}
	i.links.StartCopy(ctx, made, download.MediaVideo)
	out.DownloadTaskID = link.GetCopyTaskID(made.Vault, made.Path)
}
