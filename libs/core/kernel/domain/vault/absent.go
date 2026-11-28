package vault

import (
	"errors"
	"time"
)

var (
	// ErrFileAbsent is what a read of a file gets when the vault lists the file and its bytes are not
	// on this device.
	ErrFileAbsent = errors.New("the file is listed but its bytes are not on this device")

	// ErrFilePresent is what an absent entry gets when a file with other content is at its path.
	ErrFilePresent = errors.New("a file with other content is at the path")

	// ErrChanged is what an eviction gets when the file does not have the hash it was asked to drop.
	ErrChanged = errors.New("the file has changes that are not the content named")
)

// AbsentFile is a file the vault lists whose bytes are not on this device. Provider is the id of the
// plugin that can fill it.
type AbsentFile struct {
	Path     string
	Size     int64
	Mtime    time.Time
	Hash     string
	Provider string
}
