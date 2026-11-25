package machine

import (
	"fmt"
	"os"
)

// UserFolders are the folders of the person on this machine.
type UserFolders struct{}

// GetHomeFolder returns the home folder of the person.
func (UserFolders) GetHomeFolder() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find the home folder: %w", err)
	}
	return home, nil
}

// GetDocumentsFolder returns the Documents folder of the person as the system names it.
func (u UserFolders) GetDocumentsFolder() (string, error) {
	home, err := u.GetHomeFolder()
	if err != nil {
		return "", err
	}
	return readPlatformDocuments(home), nil
}
