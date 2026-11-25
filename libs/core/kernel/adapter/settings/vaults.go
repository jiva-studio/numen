package settings

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
