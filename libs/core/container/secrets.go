package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// credentialsLocator is a key store that keeps its values in a file when the machine has no keychain.
type credentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}

// openSecretLedger returns the list of secrets in the settings file.
func (c Config) openSecretLedger() *settings.SecretLedger {
	return settings.NewSecretLedger(c.getResources().settingsFiles, c.settingsFile, c.readCheckedSettings, c.refreshSettingEvents)
}

// OpenSecrets returns the use cases on the secrets shared between plugins.
func (c Config) OpenSecrets() *secret.Secrets {
	var plugins pluginport.SecretPlugins = secret.NoPlugins{}
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	return secret.NewSecrets(c.openSecretLedger(), c.KeyStore, plugins, newSecretID, c.getCredentialsPath)
}

// getCredentialsPath returns the file the values of secrets are kept in, and the empty text while
// the system keychain is in use.
func (c Config) getCredentialsPath() string {
	locator, isLocator := c.KeyStore.(credentialsLocator)
	if !isLocator {
		return ""
	}
	return locator.GetCredentialsPath(context.Background())
}

// newSecretID makes the id of a secret: "s" and 8 random hexadecimal digits.
func newSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}

// setSecretSource gives the plugin host the list of secrets and the means to drop permissions.
func (c Config) setSecretSource() {
	ledger := c.openSecretLedger()
	c.PluginHost.SetSecretSource(pluginport.SecretSource{
		List:  ledger.GetSecrets,
		Prune: c.OpenSecrets().Prune.Execute,
	})
}
