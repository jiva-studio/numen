package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

var (
	// ErrHashMismatch is what a placed file gets when the staged content is not the expected one.
	ErrHashMismatch = errors.New("the staged file does not have the expected hash")
	// ErrInvalidRange is what a ranged read gets for a length outside 1 to MaxReadBytes.
	ErrInvalidRange = errors.New("the range is invalid")
)

const MaxReadBytes = 4 << 20

// MoveFileResult is the count of files under the new path.
type MoveFileResult struct{ Count int }

// RemoveFileResult is the count of files moved to the trash.
type RemoveFileResult struct{ Count int }

// ByteRange is bytes of a file as they are, with the stat the file had when they were read.
type ByteRange struct {
	Content []byte
	Stat    domain.FileStat
}

// FileMover files a file of any kind, or a folder with everything under it, somewhere else, or in the trash.
// A baseline that names a file the caller last saw yields ErrStale when the file changed; a folder has none.
type FileMover interface {
	MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (MoveFileResult, error)
	RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (RemoveFileResult, error)
}

// FileRangeReader reads bytes of any file as they are.
type FileRangeReader interface {
	ReadBytes(ctx context.Context, path string, start, length int64) (ByteRange, error)
}

// FilePlacer moves a file of the plugin's own store folder to a path of the vault in one atomic rename.
type FilePlacer interface {
	PlaceFile(ctx context.Context, name, path, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error)
}

var (
	_ FileMover       = (*Client)(nil)
	_ FileRangeReader = (*Client)(nil)
	_ FilePlacer      = (*Client)(nil)
)

func (c *Client) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (MoveFileResult, error) {
	res, err := c.files.MoveFile(ctx, request(c, &v1.HostFileServiceMoveFileRequest{From: from, To: to, Baseline: writeFileStat(readBaseline(baseline))}))
	if err != nil {
		return MoveFileResult{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return MoveFileResult{}, err
	}
	return MoveFileResult{Count: int(res.Msg.GetMovedCount())}, nil
}

func (c *Client) RemoveFile(ctx context.Context, path string, baseline *domain.FileStat) (RemoveFileResult, error) {
	res, err := c.files.RemoveFile(ctx, request(c, &v1.HostFileServiceRemoveFileRequest{Path: path, Baseline: writeFileStat(readBaseline(baseline))}))
	if err != nil {
		return RemoveFileResult{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return RemoveFileResult{}, err
	}
	return RemoveFileResult{Count: int(res.Msg.GetRemovedCount())}, nil
}

func (c *Client) ReadBytes(ctx context.Context, path string, start, length int64) (ByteRange, error) {
	res, err := c.files.ReadFileRange(ctx, request(c, &v1.ReadFileRangeRequest{Path: path, Start: start, Length: length}))
	if err != nil {
		return ByteRange{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return ByteRange{}, err
	}
	return ByteRange{Content: res.Msg.GetContent(), Stat: readFileStat(res.Msg.GetStat())}, nil
}

func (c *Client) PlaceFile(ctx context.Context, name, path, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error) {
	res, err := c.files.PlaceFile(ctx, request(c, &v1.PlaceFileRequest{
		Name: name, Path: path, ExpectedHash: expectedHash, Baseline: writeFileStat(readBaseline(baseline)),
	}))
	if err != nil {
		return domain.FileStat{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.FileStat{}, err
	}
	return readFileStat(res.Msg.GetStat()), nil
}

func readBaseline(baseline *domain.FileStat) domain.FileStat {
	if baseline == nil {
		return domain.FileStat{}
	}
	return *baseline
}
