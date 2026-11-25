package settings

import (
	"fmt"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// Vaults is the settings section of the vaults of this installation: Folder is where the folders of new
// vaults are made, and empty means the Documents folder of the system.
type Vaults struct {
	Folder string `json:"folder"`
}

// Texts holds the string settings of the shell that stand alone in their section of the file, by their
// declared key: logging.level, keys.credentialsFile and vaults.folder.
type Texts map[string]string

// DefaultTexts returns the texts the declaration of the shell gives.
func DefaultTexts() Texts {
	return Texts{
		keyLogLevel:        getAppString(keyLogLevel),
		keyCredentialsFile: getAppString(keyCredentialsFile),
		keyVaultsFolder:    getAppString(keyVaultsFolder),
	}
}

// GetNewVaultsFolder returns the setting Folder for new vaults of the file, or "" when it is not text.
func GetNewVaultsFolder(file File) string {
	value, _ := file.GetValue(settingsdomain.AppOwner, "vaults", "folder")
	text, isText := value.(string)
	if !isText {
		return ""
	}
	return text
}

// NewNewVaultsFolderReader creates the reading of the setting Folder for new vaults. It reads the settings on
// each call and reports a failed read to handle.
func NewNewVaultsFolderReader(read func() (File, error), handle func(error)) func() string {
	return func() string {
		file, err := read()
		if err != nil {
			handle(fmt.Errorf("read settings: %w", err))
		}
		return GetNewVaultsFolder(file)
	}
}
