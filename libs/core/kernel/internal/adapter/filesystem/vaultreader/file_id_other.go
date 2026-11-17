//go:build !unix

package vaultreader

import (
	"io/fs"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func getFileID(fs.FileInfo) domain.FileID { return 0 }
