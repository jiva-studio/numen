//go:build unix

package vaultreader

import (
	"io/fs"
	"syscall"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func getFileID(info fs.FileInfo) domain.FileID {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}
	return domain.FileID(stat.Ino)
}
