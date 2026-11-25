package settings

// Vaults is the settings section of the vaults of this installation: Folder is where the folders of new
// vaults are made, and empty means the Documents folder of the system.
type Vaults struct {
	Folder string `json:"folder"`
}

// DefaultVaults is the section of an unconfigured installation.
func DefaultVaults() Vaults { return Vaults{Folder: getAppString(keyVaultsFolder)} }

// GetNewVaultsFolder returns the folder set for new vaults, and an empty text when none is set.
func (f *File) GetNewVaultsFolder() string { return f.Vaults.Folder }
