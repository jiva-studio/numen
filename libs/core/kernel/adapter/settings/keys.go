package settings

import (
	"errors"
	"io/fs"
	"os"
	"slices"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// The values of keys.credentialsFile.
const (
	CredentialsFileUnset      = "unset"
	CredentialsFilePassphrase = "passphrase"
	CredentialsFilePlaintext  = "plaintext"
)

// Keys is the settings section for where the keys are kept on a computer without a system keychain.
type Keys struct {
	CredentialsFile string `json:"credentialsFile"`
}

// DefaultKeys is the section of an unconfigured installation.
func DefaultKeys() Keys { return Keys{CredentialsFile: getAppString(keyCredentialsFile)} }

// ReadCredentialsFile reads keys.credentialsFile from the settings file at path without writing
// anything. A file that is not there, cannot be read or holds another value gives CredentialsFileUnset.
func ReadCredentialsFile(path string) string {
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return CredentialsFileUnset
	}
	file, err := (&Store{files: NewFiles()}).ReadUncheckedAt(path)
	if err != nil {
		return CredentialsFileUnset
	}
	value := file.Keys.CredentialsFile
	if !slices.Contains([]string{CredentialsFilePassphrase, CredentialsFilePlaintext}, value) {
		return CredentialsFileUnset
	}
	return value
}

// WriteCredentialsFile keeps value as keys.credentialsFile in the settings file at path.
func WriteCredentialsFile(path, value string) error {
	store := NewStore(NewFiles(), func() (string, error) { return path, nil }, nil, nil)
	return store.Write([]settingsdomain.Setting{{
		Path: []string{settingsdomain.AppOwner, "keys", "credentialsFile"},
		JSON: `"` + value + `"`,
	}})
}
