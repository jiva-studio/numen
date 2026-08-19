package filesystem

import (
	"golang.org/x/sys/windows"
)

// documents is the folder Windows keeps documents in, asked of Windows.
//
// It is not under the profile whenever somebody has moved it, which a machine
// signed in to OneDrive has done for them.
func documents() (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Documents, 0)
}
