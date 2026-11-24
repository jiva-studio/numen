// Package downloadwire answers the methods that download an absent file of a vault, drop that download and list the
// downloads that run.
package downloadwire

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/dropdownload"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
)

const lengthPrefix = 8

// Deps are what the methods need.
type Deps struct {
	Fetch   fetchfile.Deps
	Entries ports.DownloadEntries
}

// Downloads answers the methods of the downloads.
type Downloads struct {
	fetch   *fetchfile.FetchFile
	drop    *dropdownload.DropDownload
	entries ports.DownloadEntries
	host    ports.Host
}

// New returns the methods over the dependencies.
func New(deps Deps) *Downloads {
	return &Downloads{
		fetch:   fetchfile.NewFetchFile(deps.Fetch),
		drop:    dropdownload.NewDropDownload(dropdownload.Deps{Host: deps.Fetch.Host, State: deps.Fetch.State}),
		entries: deps.Entries,
		host:    deps.Fetch.Host,
	}
}

// FetchFile downloads the absent file at the path of the vault.
func (d *Downloads) FetchFile(ctx context.Context, request api.FetchFileRequest) (api.FetchFileResponse, error) {
	return api.FetchFileResponse{}, d.fetch.Execute(ctx, domain.VaultID(request.VaultID), request.Path)
}

// DropDownload removes the download of the file at the path of the vault.
func (d *Downloads) DropDownload(ctx context.Context, request api.DropDownloadRequest) (api.DropDownloadResponse, error) {
	return api.DropDownloadResponse{}, d.drop.Execute(ctx, domain.VaultID(request.VaultID), request.Path)
}

// GetDownloads lists the downloads that were started and not finished, with the bytes that came and the size of
// each file.
func (d *Downloads) GetDownloads(ctx context.Context, _ api.GetDownloadsRequest) (api.GetDownloadsResponse, error) {
	vaults, err := d.host.GetVaults(ctx)
	if err != nil {
		return api.GetDownloadsResponse{Downloads: nil}, err
	}
	listed := []api.Download{}
	for _, vault := range vaults {
		kept, err := d.getVaultDownloads(ctx, vault.ID)
		if err != nil {
			return api.GetDownloadsResponse{Downloads: nil}, err
		}
		listed = append(listed, kept...)
	}
	return api.GetDownloadsResponse{Downloads: listed}, nil
}

func (d *Downloads) getVaultDownloads(ctx context.Context, id domain.VaultID) ([]api.Download, error) {
	entries, err := d.entries.ReadDownloads(ctx, id)
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	absent, err := d.host.GetAbsent(id).ListAbsentFiles(ctx)
	if err != nil {
		return nil, err
	}
	sizes := map[string]int64{}
	for _, file := range absent {
		sizes[file.Path] = file.Size
	}
	listed := make([]api.Download, 0, len(entries))
	for _, entry := range entries {
		listed = append(listed, api.Download{VaultID: string(id), Path: entry.Path, Bytes: getBytes(entry.Header, entry.NextChunk), Total: sizes[entry.Path]})
	}
	return listed, nil
}

func getBytes(header []byte, next uint64) int64 {
	parsed, err := blobcipher.ParseHeader(header)
	if err != nil || next == 0 {
		return 0
	}
	return int64(next)<<parsed.ChunkLog2 - lengthPrefix
}
