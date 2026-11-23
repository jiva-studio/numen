package index

import (
	"errors"
	"os"
)

// OpenFailure is the reason an index file could not be opened.
type OpenFailure int

const (
	// NotAnIndex is a file that is not a database.
	NotAnIndex OpenFailure = iota + 1
	// PathIsFolder is an index path that names a folder.
	PathIsFolder
	// CannotCreate is a file the database could neither open nor make.
	CannotCreate
	// OtherOpenFailure is any other result code of the database.
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
