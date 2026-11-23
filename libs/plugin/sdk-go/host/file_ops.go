package host

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	// ErrHashMismatch is what a placed file gets when the staged content is not the expected one.
	ErrHashMismatch = errors.New("the staged file does not have the expected hash")
	// ErrInvalidRange is what a ranged read gets for a length outside 1 to MaxReadBytes.
	ErrInvalidRange = errors.New("the range is invalid")
)

// MaxReadBytes is the most bytes one ReadBytes returns.
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
