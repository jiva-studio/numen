//go:build !unix

package vaultreader

import (
	"io/fs"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// getFileID returns zero: the reader reads no file id from this platform's stat.
func getFileID(fs.FileInfo) domain.FileID { return 0 }
