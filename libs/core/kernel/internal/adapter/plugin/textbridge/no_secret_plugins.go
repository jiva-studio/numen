package textbridge

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// NoSecretPlugins replaces the plugin host for the secrets where the installation has none: no
// plugin declares a secret, and an operation only changes the list.
type NoSecretPlugins struct{}

// ListDeclarations returns no declaration.
func (NoSecretPlugins) ListDeclarations() []plugin.DeclarationKey { return nil }

// Refresh configures no plugin.
func (NoSecretPlugins) Refresh(context.Context, ...string) error { return nil }
