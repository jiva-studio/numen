package index

import (
	"errors"
	"os"
)

// OpenFailure is the reason an index file could not be opened. CannotCreate is a file sqlite could
// neither open nor make.
type OpenFailure int

const (
	NotAnIndex OpenFailure = iota + 1
	PathIsFolder
	CannotCreate
	OtherOpenFailure
)

// The result codes sqlite answers with.
const (
	codeNotADatabase = 26
	codeCannotOpen   = 14
)

// GetOpenFailure classifies an error returned when opening the index at path. It
// returns false when the error carries no sqlite result code.
func GetOpenFailure(path string, why error) (OpenFailure, bool) {
	var coded interface{ Code() int }
	if !errors.As(why, &coded) {
		return 0, false
	}
	switch code := coded.Code(); {
	case code == codeNotADatabase:
		return NotAnIndex, true
	case code == codeCannotOpen && isFolder(path):
		return PathIsFolder, true
	case code == codeCannotOpen:
		return CannotCreate, true
	default:
		return OtherOpenFailure, true
	}
}

func isFolder(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
