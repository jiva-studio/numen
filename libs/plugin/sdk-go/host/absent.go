package host

import (
	"context"
	"errors"
	"time"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var (
	// ErrFileAbsent is what a read gets for a file the vault lists and does not hold.
	ErrFileAbsent = errors.New("the file is listed but its bytes are not on this device")
	// ErrFilePresent is what listing a file as absent gets when a file with other content is at its path.
	ErrFilePresent = errors.New("a file with other content is at the path")
	// ErrHashMismatch is what a fill gets when the bytes do not have the hash of the entry.
	ErrHashMismatch = errors.New("the bytes do not have the hash of the entry")
	// ErrChanged is what an eviction gets when the file does not have the hash it was asked to drop.
	ErrChanged = errors.New("the file has changes that are not the content named")
)

// AbsentFile is a file the vault lists whose bytes are not on this device. Hash is the SHA-256 of the
// content, hex.
type AbsentFile struct {
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

// AbsentFiles lists files the device does not hold and fills them later. A plugin that syncs a vault
// to a device with little room uses it.
type AbsentFiles interface {
	// PutAbsentFiles lists the files, replacing the entry of each path. A path that holds a file of
	// another hash returns ErrFilePresent and none is listed.
	PutAbsentFiles(ctx context.Context, files []AbsentFile) error
	// ListAbsentFiles returns the files listed.
	ListAbsentFiles(ctx context.Context) ([]AbsentFile, error)
	// FillFile places the file storeName of the plugin's store folder at the path of a listed file. A
	// file whose SHA-256 is not the hash of the entry returns ErrHashMismatch and nothing is placed. The
	// fill is no change the feed reports.
	FillFile(ctx context.Context, path, storeName string) error
	// EvictFile deletes the bytes of the file and lists it, when its SHA-256 is expectedHash, else it
	// returns ErrChanged. The eviction is no removal the feed reports.
	EvictFile(ctx context.Context, path, expectedHash string) error
}

var _ AbsentFiles = (*Client)(nil)

func (c *Client) PutAbsentFiles(ctx context.Context, files []AbsentFile) error {
	sent := make([]*v1.HostAbsentFile, len(files))
	for i, file := range files {
		sent[i] = &v1.HostAbsentFile{Path: file.Path, Size: file.Size, ModifiedAtMs: file.ModTime.UnixMilli(), Hash: file.Hash}
	}
	res, err := c.vaults.PutAbsentFiles(ctx, request(c, &v1.PutAbsentFilesRequest{Files: sent}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) ListAbsentFiles(ctx context.Context) ([]AbsentFile, error) {
	res, err := c.vaults.ListAbsentFiles(ctx, request(c, &v1.ListAbsentFilesRequest{}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	listed := make([]AbsentFile, len(res.Msg.GetFiles()))
	for i, file := range res.Msg.GetFiles() {
		listed[i] = AbsentFile{Path: file.GetPath(), Size: file.GetSize(), ModTime: time.UnixMilli(file.GetModifiedAtMs()), Hash: file.GetHash()}
	}
	return listed, nil
}

func (c *Client) FillFile(ctx context.Context, path, storeName string) error {
	res, err := c.vaults.FillFile(ctx, request(c, &v1.FillFileRequest{Path: path, StoreName: storeName}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) EvictFile(ctx context.Context, path, expectedHash string) error {
	res, err := c.vaults.EvictFile(ctx, request(c, &v1.EvictFileRequest{Path: path, ExpectedHash: expectedHash}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}
