package settings

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// SecretLedger is the list of secrets in the settings file.
type SecretLedger struct {
	files   *Files
	path    func() (string, error)
	read    func() (File, error)
	changed func()
}

// NewSecretLedger returns the list of secrets in the file at path. read answers the settings as they
// are in force, and changed is called after each write.
func NewSecretLedger(files *Files, path func() (string, error), read func() (File, error), changed func()) *SecretLedger {
	return &SecretLedger{files: files, path: path, read: read, changed: changed}
}

// GetSecrets returns the valid secrets of the file in list order.
func (l *SecretLedger) GetSecrets() ([]plugin.Secret, error) {
	current, err := l.read()
	if err != nil {
		return nil, err
	}
	return current.Secrets, nil
}

// ChangeSecrets reads the secrets section, applies change and writes it back under the lock of the
// settings file, and delivers the change to the readers of the setting-changed event.
func (l *SecretLedger) ChangeSecrets(change func([]plugin.Secret) ([]plugin.Secret, error)) error {
	path, err := l.path()
	if err != nil {
		return err
	}
	if err := l.files.ChangeSecrets(path, change); err != nil {
		return err
	}
	l.changed()
	return nil
}

// CreateSecretID returns the id of a new secret: the letter s and eight hexadecimal digits from the
// system's random source.
func CreateSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}
