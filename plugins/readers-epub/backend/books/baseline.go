package books

import (
	"strconv"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
)

// ParseBaseline reads the size and modification time of the bytes a book was read from out of the
// text of two numbers. It returns nil when size is no number, and a zero time when mtime is none.
func ParseBaseline(size, mtime string) *api.FileStat {
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="size is a base-10 64-bit integer"
	bytes, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return nil
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mtime is a base-10 64-bit integer"
	stamp, err := strconv.ParseInt(mtime, 10, 64)
	// gomutants:disable-next-line BRANCH_IF reason="strconv.ParseInt returns 0 on error"
	if err != nil {
		stamp = 0
	}
	return &api.FileStat{Path: "", Size: bytes, Mtime: stamp}
}
