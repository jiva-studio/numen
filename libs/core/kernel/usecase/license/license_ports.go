package license

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ChangeHub broadcasts license state changes to watchers and pushes updated configuration to
// running plugins.
type ChangeHub interface {
	// NotifyChange signals that a license state changed.
	NotifyChange(ctx context.Context)
	// Subscribe returns a channel that receives a signal whenever license state or plugins change,
	// and a function to stop listening.
	Subscribe() (<-chan struct{}, func())
}

// Declarations lists the plugins that can declare licenses.
type Declarations interface {
	// ListPlugins returns all discovered plugins registered with the host, in the order of their IDs.
	ListPlugins() []manifest.DiscoveredPlugin
}

// MachineIdentity provides the current machine's fingerprint and human-readable instance name.
type MachineIdentity interface {
	// GetMachineFingerprint returns the deterministic SHA-256 fingerprint of this machine.
	GetMachineFingerprint() string
	// GetInstanceName returns the human-readable name of this machine sent to licensing providers.
	GetInstanceName() string
}

// Validator activates, validates and deactivates machine license keys across the providers a
// license declares, and resolves their checkout options.
type Validator interface {
	// ListProviderOptions resolves the display names, checkout URLs and payment methods for the
	// providers a license declares.
	ListProviderOptions(providers []manifest.LicenseProvider) []license.ProviderOption
	// Activate tries the declared providers in order and returns the activation result or the most
	// specific failure.
	Activate(
		ctx context.Context,
		providers []manifest.LicenseProvider,
		key, instanceName string,
	) (license.ActivationResult, error)
	// Validate checks whether a license key and machine instance remain valid at the named provider.
	Validate(
		ctx context.Context,
		provider, key, instanceID string,
	) (license.ValidationResult, error)
	// Deactivate releases a machine instance slot at the named provider.
	Deactivate(ctx context.Context, provider, key, instanceID string) error
}
