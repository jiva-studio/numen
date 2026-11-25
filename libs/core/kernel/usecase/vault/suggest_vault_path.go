package vault

import (
	"fmt"
	"path/filepath"
	"strings"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NewVaultsFolder is the value of the setting Folder for new vaults. It is empty when the person set none.
type NewVaultsFolder func() string

// SuggestedPath is where a new vault of a name goes.
type SuggestedPath struct {
	Folder  string
	Path    string
	IsTaken bool
}

// SuggestVaultPath names the folder of a new vault.
type SuggestVaultPath struct {
	setting NewVaultsFolder
	user    UserFolders
	folders NewVaultFolders
}

// NewSuggestVaultPath creates the suggestion over the setting, the folders of the person and the filesystem.
func NewSuggestVaultPath(setting NewVaultsFolder, user UserFolders, folders NewVaultFolders) *SuggestVaultPath {
	return &SuggestVaultPath{setting: setting, user: user, folders: folders}
}

// Execute returns the folder given, or the setting Folder for new vaults when none is given, or the Documents
// folder when the setting is empty, and the name in it. A name that is taken there gets the lowest free number
// from 2 appended.
func (u *SuggestVaultPath) Execute(name, chosen string) (SuggestedPath, error) {
	folder := strings.TrimSpace(chosen)
	if folder == "" {
		folder = strings.TrimSpace(u.setting())
	}
	if folder == "" {
		documents, err := u.user.GetDocumentsFolder()
		if err != nil {
			return SuggestedPath{}, err
		}
		folder = documents
	}
	if !domain.IsAbsPath(folder) {
		return SuggestedPath{}, newInvalidPath("the folder for new vaults %q is not an absolute path", folder)
	}
	folder = filepath.Clean(folder)
	base := getFolderName(name)
	suggested := SuggestedPath{Folder: folder, Path: filepath.Join(folder, base)}
	for number := 1; ; number++ {
		if number > 1 {
			suggested.Path = filepath.Join(folder, fmt.Sprintf("%s %d", base, number))
		}
		state, err := u.folders.GetFolderState(suggested.Path)
		if err != nil {
			return SuggestedPath{}, err
		}
		if !state.Exists {
			return suggested, nil
		}
		suggested.IsTaken = true
	}
}

func getFolderName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '-'
		}
		return r
	}, name)
	cleaned = strings.Trim(cleaned, " .")
	if cleaned == "" {
		return vaultdomain.DefaultFolderName
	}
	return cleaned
}
