package pluginport

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// SecretLedger is the ordered list of secrets kept in the settings file.
type SecretLedger interface {
	// GetSecrets returns the valid secrets in list order.
	GetSecrets() ([]plugin.Secret, error)
	// ChangeSecrets reads the list, applies change and writes the result as one step. Nothing is
	// written when change fails or leaves the list as it was.
	ChangeSecrets(change func([]plugin.Secret) ([]plugin.Secret, error)) error
}

// SecretPlugins is what the use cases on secrets need from the plugin host.
type SecretPlugins interface {
	// ListDeclarations returns every secret declaration of every installed plugin.
	ListDeclarations() []plugin.DeclarationKey
	// Refresh configures each running plugin whose list of secrets differs from the one it holds,
	// and each plugin named in forced although its list did not change, and waits for every send.
	// An operation calls it after its change is committed, with the plugins that read the secret it
	// wrote. The error names the plugin whose send failed.
	Refresh(ctx context.Context, forced ...string) error
}
