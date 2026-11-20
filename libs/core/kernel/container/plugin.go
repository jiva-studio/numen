package container

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ConfigRefresher sends the configuration again to the plugins that were not given it yet.
type ConfigRefresher interface {
	// Refresh configures the running plugins that were not given their configuration yet, and the
	// plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
}

// PluginBinding gives the host the mode and the vault path the plugins run under.
type PluginBinding interface {
	SetSafeMode(enabled bool)
	SetVaultPath(path string)
}

type PluginEventEmitter interface {
	EmitEvent(event string)
	// EmitFilesChanged emits the onFilesChanged event of the vault to the plugins that take a file of one of
	// the types.
	EmitFilesChanged(vaultID string, types []domain.FileType)
}

// PluginRetrier starts a plugin once, as any use of it does.
type PluginRetrier interface {
	// RetryPlugin answers why the start failed where it did.
	RetryPlugin(ctx context.Context, pluginID string) (*plugin.Failure, error)
}

type PluginSettingDeclarer interface {
	// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and
	// false for a plugin the host does not know.
	GetPluginSettingDeclarations(pluginID string) ([]manifest.Setting, bool)
}

// PluginShutdown stops the plugins when the host shuts down.
type PluginShutdown interface {
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}
