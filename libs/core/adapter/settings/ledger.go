package settings

import "github.com/jiva-studio/numen/libs/core/port/pluginport"

// SecretLedger is the list of secrets in the settings file.
type SecretLedger struct {
	files *Files
	// path answers where the settings file is.
	path func() (string, error)
	// read answers the settings with the invalid secrets left out.
	read func() (File, error)
	// changed delivers a write to the readers of the setting-changed event.
	changed func()
}

// NewSecretLedger returns the list of secrets in the file at path. read answers the settings as they
// are in force, and changed is called after each write.
func NewSecretLedger(files *Files, path func() (string, error), read func() (File, error), changed func()) *SecretLedger {
	return &SecretLedger{files: files, path: path, read: read, changed: changed}
}

// GetSecrets returns the valid secrets of the file in list order.
func (l *SecretLedger) GetSecrets() ([]pluginport.Secret, error) {
	current, err := l.read()
	if err != nil {
		return nil, err
	}
	return current.Secrets, nil
}

// ChangeSecrets reads the secrets section, applies change and writes it back under the lock of the
// settings file, and delivers the change to the readers of the setting-changed event.
func (l *SecretLedger) ChangeSecrets(change func([]pluginport.Secret) ([]pluginport.Secret, error)) error {
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
