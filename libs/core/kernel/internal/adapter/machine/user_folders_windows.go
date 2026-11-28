//go:build windows

package machine

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

func readPlatformDocuments(home string) string {
	documents, err := windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return filepath.Join(home, "Documents")
	}
	return documents
}
