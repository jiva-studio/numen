package host

import (
	"context"
	"errors"
	"time"
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

	errAbsentNotImplemented = errors.New("not implemented")
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

func (c *Client) PutAbsentFiles(context.Context, []AbsentFile) error { return errAbsentNotImplemented }

func (c *Client) ListAbsentFiles(context.Context) ([]AbsentFile, error) {
	return nil, errAbsentNotImplemented
}

func (c *Client) FillFile(context.Context, string, string) error { return errAbsentNotImplemented }

func (c *Client) EvictFile(context.Context, string, string) error { return errAbsentNotImplemented }
